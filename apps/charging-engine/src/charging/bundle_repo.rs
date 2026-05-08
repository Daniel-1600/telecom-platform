use sqlx::postgres::{PgPool, PgPoolOptions};
use sqlx::Row;
use crate::circuit_breaker::{CircuitBreaker, CircuitBreakerError};
use crate::errors::{ChargingError, ChargingResult};
use super::types::{Bundle, BundleType, BundleAllowances, ActiveBundle};
use chrono::Utc;

#[derive(Clone)]
pub struct BundleRepo {
    pub(crate) pool: PgPool,
    circuit_breaker: CircuitBreaker,
}

impl BundleRepo {
    pub async fn connect(database_url: &str) -> ChargingResult<Self> {
        let pool = PgPoolOptions::new()
            .max_connections(10)
            .connect(database_url)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;
        let circuit_breaker = CircuitBreaker::new(5, std::time::Duration::from_secs(60));
        Ok(Self { pool, circuit_breaker })
    }

    /// Upsert a bundle definition.
    pub async fn upsert_bundle(&self, bundle: &Bundle) -> ChargingResult<()> {
        let bundle = bundle.clone();
        self.circuit_breaker.execute(|| async move {
            sqlx::query(r#"
                INSERT INTO bundles
                    (bundle_id, name, bundle_type, data_bytes, voice_seconds,
                     sms_count, roaming_data_bytes, validity_days, priority, is_active, updated_at)
                VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,TRUE,NOW())
                ON CONFLICT (bundle_id) DO UPDATE SET
                    name = EXCLUDED.name,
                    bundle_type = EXCLUDED.bundle_type,
                    data_bytes = EXCLUDED.data_bytes,
                    voice_seconds = EXCLUDED.voice_seconds,
                    sms_count = EXCLUDED.sms_count,
                    roaming_data_bytes = EXCLUDED.roaming_data_bytes,
                    validity_days = EXCLUDED.validity_days,
                    priority = EXCLUDED.priority,
                    is_active = TRUE,
                    updated_at = NOW()
            "#)
            .bind(&bundle.bundle_id)
            .bind(&bundle.name)
            .bind(bundle.bundle_type.as_str())
            .bind(bundle.allowances.data_bytes.map(|v| v as i64))
            .bind(bundle.allowances.voice_seconds.map(|v| v as i64))
            .bind(bundle.allowances.sms_count.map(|v| v as i64))
            .bind(bundle.allowances.roaming_data_bytes.map(|v| v as i64))
            .bind(bundle.validity_days as i32)
            .bind(bundle.priority as i16)
            .execute(&self.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;
            Ok(())
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    pub async fn get_bundle(&self, bundle_id: &str) -> ChargingResult<Option<Bundle>> {
        let bundle_id = bundle_id.to_string();
        self.circuit_breaker.execute(|| async move {
            let row = sqlx::query(r#"
                SELECT bundle_id, name, bundle_type, data_bytes, voice_seconds,
                       sms_count, roaming_data_bytes, validity_days, priority, is_active
                FROM bundles WHERE bundle_id = $1 AND is_active = TRUE
            "#)
            .bind(&bundle_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;
            Ok(row.map(row_to_bundle))
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    /// Activate a bundle for a subscriber — inserts into subscriber_bundles.
    /// Returns the new subscriber_bundle id.
    pub async fn activate_for_subscriber(
        &self,
        imsi: &str,
        bundle_id: &str,
        validity_days: u32,
        allowances: &BundleAllowances,
    ) -> ChargingResult<i64> {
        let imsi = imsi.to_string();
        let bundle_id = bundle_id.to_string();
        let expires_at = Utc::now() + chrono::Duration::days(validity_days as i64);
        let allowances = allowances.clone();

        self.circuit_breaker.execute(|| async move {
            let row = sqlx::query(r#"
                INSERT INTO subscriber_bundles
                    (imsi, bundle_id, expires_at,
                     remaining_data_bytes, remaining_voice_seconds,
                     remaining_sms_count, remaining_roaming_bytes, status)
                VALUES ($1,$2,$3,$4,$5,$6,$7,'active')
                RETURNING id
            "#)
            .bind(&imsi)
            .bind(&bundle_id)
            .bind(expires_at)
            .bind(allowances.data_bytes.map(|v| v as i64))
            .bind(allowances.voice_seconds.map(|v| v as i64))
            .bind(allowances.sms_count.map(|v| v as i64))
            .bind(allowances.roaming_data_bytes.map(|v| v as i64))
            .fetch_one(&self.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

            Ok(row.get::<i64, _>("id"))
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    /// Fetch active, non-expired bundles for a subscriber, ordered by priority.
    pub async fn get_active_bundles(&self, imsi: &str) -> ChargingResult<Vec<ActiveBundle>> {
        let imsi = imsi.to_string();
        self.circuit_breaker.execute(|| async move {
            let rows = sqlx::query(r#"
                SELECT sb.id, sb.imsi, sb.bundle_id, sb.activated_at, sb.expires_at,
                       sb.remaining_data_bytes, sb.remaining_voice_seconds,
                       sb.remaining_sms_count, sb.remaining_roaming_bytes,
                       b.priority
                FROM subscriber_bundles sb
                JOIN bundles b ON b.bundle_id = sb.bundle_id
                WHERE sb.imsi = $1
                  AND sb.status = 'active'
                  AND sb.expires_at > NOW()
                ORDER BY b.priority ASC, sb.activated_at ASC
            "#)
            .bind(&imsi)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

            Ok(rows.into_iter().map(row_to_active_bundle).collect())
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    /// Atomically deduct from a subscriber_bundle row.
    /// Returns true if deduction succeeded, false if insufficient allowance.
    pub async fn deduct_from_bundle(
        &self,
        subscriber_bundle_id: i64,
        usage_type: &str,
        amount: u64,
    ) -> ChargingResult<bool> {
        let amount = amount as i64;
        self.circuit_breaker.execute(|| async move {
            let column = match usage_type {
                "data"    => "remaining_data_bytes",
                "voice"   => "remaining_voice_seconds",
                "sms"     => "remaining_sms_count",
                "roaming" => "remaining_roaming_bytes",
                _ => return Err(ChargingError::InvalidInput(format!("Unknown usage type: {}", usage_type))),
            };

            // Atomic check-and-deduct: only update if remaining >= amount
            let result = sqlx::query(&format!(r#"
                UPDATE subscriber_bundles
                SET {column} = {column} - $1,
                    status = CASE
                        WHEN ({column} - $1) <= 0 THEN 'exhausted'
                        ELSE status
                    END,
                    updated_at = NOW()
                WHERE id = $2
                  AND {column} >= $1
                  AND status = 'active'
                  AND expires_at > NOW()
            "#))
            .bind(amount)
            .bind(subscriber_bundle_id)
            .execute(&self.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

            Ok(result.rows_affected() > 0)
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    /// Expire bundles whose expires_at has passed — run this from a background task.
    pub async fn expire_stale_bundles(&self) -> ChargingResult<u64> {
        self.circuit_breaker.execute(|| async {
            let result = sqlx::query(r#"
                UPDATE subscriber_bundles
                SET status = 'expired', updated_at = NOW()
                WHERE status = 'active' AND expires_at <= NOW()
            "#)
            .execute(&self.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;
            Ok(result.rows_affected())
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }
}
