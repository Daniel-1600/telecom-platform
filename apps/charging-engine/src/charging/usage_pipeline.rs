use chrono::Utc;
use tracing::{info, warn};

use crate::errors::{ChargingError, ChargingResult};
use super::types::{UsageEvent, UsageType};
use super::engine::ChargingEngine;

#[derive(Debug)]
pub enum ChargingSource {
    Bundle { subscriber_bundle_id: i64 },
    Credit,
}

pub struct ChargeResult {
    pub imsi: String,
    pub session_id: String,
    pub volume: u64,
    pub cost: f64,
    pub charging_source: ChargingSource,
}

impl ChargingEngine {
    /// Full charging pipeline: bundle-first, then credit fallback.
    /// This is the method called by the `process_usage` handler.
    pub async fn process_usage_event(&self, event: UsageEvent) -> ChargingResult<ChargeResult> {
        // 1. Validate subscriber exists and is active
        let subscriber = self.get_subscriber_account(&event.imsi).await?
            .ok_or_else(|| ChargingError::SubscriberNotFound(event.imsi.clone()))?;

        if !matches!(subscriber.status, crate::charging::types::AccountStatus::Active) {
            return Err(ChargingError::UsageBlocked(
                format!("Subscriber {} is not active", event.imsi)
            ));
        }

        let usage_type_str = match event.usage_type {
            UsageType::Data  => "data",
            UsageType::Voice => "voice",
            UsageType::SMS   => "sms",
        };

        // 2. Try bundle deduction first
        let active_bundles = self.bundle_repo.get_active_bundles(&event.imsi).await?;

        for bundle in &active_bundles {
            let has_allowance = match event.usage_type {
                UsageType::Data  => bundle.remaining_allowances.data_bytes
                    .map_or(false, |b| b >= event.volume),
                UsageType::Voice => bundle.remaining_allowances.voice_seconds
                    .map_or(false, |s| s >= event.volume),
                UsageType::SMS   => bundle.remaining_allowances.sms_count
                    .map_or(false, |c| c >= event.volume),
            };

            if !has_allowance {
                continue;
            }

            let deducted = self.bundle_repo
                .deduct_from_bundle(bundle.id, usage_type_str, event.volume)
                .await?;

            if deducted {
                info!(
                    imsi = %event.imsi,
                    bundle_id = %bundle.bundle_id,
                    volume = event.volume,
                    usage_type = usage_type_str,
                    "Usage charged from bundle"
                );

                // Persist usage record with charging_source = 'bundle'
                self.persist_usage_record(&event, 0.0, ChargingSource::Bundle {
                    subscriber_bundle_id: bundle.id,
                }).await?;

                return Ok(ChargeResult {
                    imsi: event.imsi,
                    session_id: event.session_id,
                    volume: event.volume,
                    cost: 0.0, // bundle usage has no per-event cost
                    charging_source: ChargingSource::Bundle {
                        subscriber_bundle_id: bundle.id,
                    },
                });
            }
        }

        // 3. No bundle covered it — fall back to credit deduction
        warn!(
            imsi = %event.imsi,
            usage_type = usage_type_str,
            "No active bundle for usage, falling back to credit"
        );

        let plan = self.plans.get("basic").await? // resolve subscriber's actual plan
            .ok_or_else(|| ChargingError::RatingPlanNotFound("basic".to_string()))?;

        let cost = self.calculate_usage_cost(&event).await?;

        // Deduct from credit balance (existing credit_management logic)
        self.deduct_credit_for_usage(&event.imsi, cost).await?;

        // Persist usage record with charging_source = 'credit'
        self.persist_usage_record(&event, cost, ChargingSource::Credit).await?;

        Ok(ChargeResult {
            imsi: event.imsi,
            session_id: event.session_id,
            volume: event.volume,
            cost,
            charging_source: ChargingSource::Credit,
        })
    }

    /// Persist a usage record to Postgres for billing reconciliation.
    async fn persist_usage_record(
        &self,
        event: &UsageEvent,
        cost: f64,
        source: ChargingSource,
    ) -> ChargingResult<()> {
        let (charging_source_str, bundle_id) = match &source {
            ChargingSource::Bundle { subscriber_bundle_id } =>
                ("bundle", Some(*subscriber_bundle_id)),
            ChargingSource::Credit =>
                ("credit", None),
        };

        sqlx::query(r#"
            INSERT INTO usage_records
                (subscriber_id, session_id, usage_type, start_time, end_time,
                 volume, rate, cost, charging_source, subscriber_bundle_id, created_at, updated_at)
            SELECT s.id, $2, $3, $4, $4, $5, $6, $7, $8, $9, NOW(), NOW()
            FROM subscribers s WHERE s.imsi = $1
        "#)
        .bind(&event.imsi)
        .bind(&event.session_id)
        .bind(match event.usage_type { UsageType::Data => "data", UsageType::Voice => "voice", UsageType::SMS => "sms" })
        .bind(event.timestamp)
        .bind(event.volume as i64)
        .bind(event.rate)
        .bind(cost)
        .bind(charging_source_str)
        .bind(bundle_id)
        .execute(&self.bundle_repo.pool)
        .await
        .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

        Ok(())
    }
}
