use crate::circuit_breaker::CircuitBreakerError;
use crate::errors::{ChargingError, ChargingResult};
use super::types::{Bundle, ActiveBundle, BundleAllowances};

impl crate::charging::ChargingEngine {
    pub async fn create_bundle(&self, bundle: Bundle) -> ChargingResult<()> {
        self.postgres_circuit_breaker.execute(|| async {
            // Store bundle configuration in PostgreSQL
            let query = r#"
                INSERT INTO bundles (bundle_id, name, bundle_type, data_bytes, voice_seconds,
                                     sms_count, roaming_data_bytes, validity_days, priority,
                                     amount_unconverted, is_active)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
                ON CONFLICT (bundle_id) DO UPDATE SET
                    name               = EXCLUDED.name,
                    bundle_type        = EXCLUDED.bundle_type,
                    data_bytes         = EXCLUDED.data_bytes,
                    voice_seconds      = EXCLUDED.voice_seconds,
                    sms_count          = EXCLUDED.sms_count,
                    roaming_data_bytes = EXCLUDED.roaming_data_bytes,
                    validity_days      = EXCLUDED.validity_days,
                    priority           = EXCLUDED.priority,
                    amount_unconverted = EXCLUDED.amount_unconverted,
                    is_active          = EXCLUDED.is_active
            "#;

            let allowances_json = serde_json::to_value(&bundle.allowances)
                .map_err(|e| ChargingError::InvalidFormat(e.to_string()))?;

            sqlx::query(query)
                .bind(&bundle.bundle_id)
                .bind(&bundle.name)
                .bind(bundle.bundle_type.as_str())
                .bind(bundle.allowances.data_bytes.map(|v| v as i64))
                .bind(bundle.allowances.voice_seconds.map(|v| v as i64))
                .bind(bundle.allowances.sms_count.map(|v| v as i64))
                .bind(bundle.allowances.roaming_data_bytes.map(|v| v as i64))
                .bind(bundle.validity_days as i32)
                .bind(bundle.priority as i16)
                .bind(bundle.amount_unconverted)
                .bind(bundle.is_active)
                .execute(&self.plans.pool)
                .await
                .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;


            Ok(())
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::PostgresConnection("Circuit breaker is open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    pub async fn activate_bundle_for_subscriber(&self, imsi: &str, bundle_id: &str) -> ChargingResult<ActiveBundle> {
        self.redis_circuit_breaker.execute(|| async {
            let mut conn = self.redis_client.get_multiplexed_async_connection()
                .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

            // Get bundle configuration from PostgreSQL cache or DB
            let bundle = self.get_bundle_config(bundle_id).await?;

            // Check if bundle already active
            let active_key = format!("bundle:active:{}:{}", imsi, bundle_id);
            let is_active: bool = redis::AsyncCommands::exists(&mut conn, &active_key).await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            if is_active {
                return Err(ChargingError::BundleAlreadyActive);
            }

            // Atomic bundle activation
            let mut pipe = redis::pipe();
            pipe.atomic();

            // Set active bundle with TTL
            let expires_at = Utc::now() + chrono::Duration::days(bundle.validity_days as i64);
            let active_bundle = ActiveBundle {
                imsi: imsi.to_string(),
                bundle_id: bundle_id.to_string(),
                priority:  bundle.priority,
                activated_at: Utc::now(),
                expires_at,
                remaining_allowances: bundle.allowances.clone(),
            };

            let bundle_json = serde_json::to_string(&active_bundle)
                .map_err(|e| ChargingError::InvalidFormat(e.to_string()))?;

            pipe.set(&active_key, bundle_json)
                .expire(&active_key, bundle.validity_days as i64 * 86400);

            let set_key = format!("bundle:active:set:{}", imsi);
            pipe.sadd(&set_key, bundle_id);

            // Apply bundle allowances to subscriber balances
            if let Some(data_bytes) = bundle.allowances.data_bytes {
                let data_key = format!("bundle:data:{}", imsi);
                pipe.incr(&data_key, data_bytes as i64);
            }

            if let Some(voice_seconds) = bundle.allowances.voice_seconds {
                let voice_key = format!("bundle:voice:{}", imsi);
                pipe.incr(&voice_key, voice_seconds as i64);
            }

            if let Some(sms_count) = bundle.allowances.sms_count {
                let sms_key = format!("bundle:sms:{}", imsi);
                pipe.incr(&sms_key, sms_count as i64);
            }

            pipe.query_async(&mut conn).await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            // Record activation for audit
            self.record_bundle_activation(&mut conn, imsi, bundle_id).await?;

            Ok(active_bundle)
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::RedisConnection("Circuit breaker is open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    pub async fn consume_from_bundle(
        &self,
        imsi: &str,
        usage_type: UsageType,
        amount: u64,
    ) -> ChargingResult<bool> {
        self.redis_circuit_breaker.execute(|| async {
            let mut conn = self.redis_client.get_multiplexed_async_connection()
                .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

            // Use SMEMBERS on the tracking Set — O(N bundles per subscriber), not O(total Redis keys)
            let set_key = format!("bundle:active:set:{}", imsi);
            let bundle_ids: Vec<String> = redis::AsyncCommands::smembers(&mut conn, &set_key)
                .await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            // Fetch all active bundle JSON values in one pipeline
            let mut pipe = redis::pipe();
            for bundle_id in &bundle_ids {
                let active_key = format!("bundle:active:{}:{}", imsi, bundle_id);
                pipe.get(&active_key);
            }
            let values: Vec<Option<String>> = pipe.query_async(&mut conn)
                .await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            // Pair bundle_ids with their JSON, filter out expired/missing entries
            let mut bundles: Vec<(String, ActiveBundle)> = bundle_ids.iter()
                .zip(values.into_iter())
                .filter_map(|(id, maybe_json)| {
                    let json = maybe_json?;
                    let b: ActiveBundle = serde_json::from_str(&json).ok()?;
                    // Remove from set if expired
                    Some((id.clone(), b))
                })
                .collect();

            // Sort by priority ascending (lower number = higher priority)
            bundles.sort_by_key(|(_, b)| b.priority);

            for (bundle_id, mut active_bundle) in bundles {
                // Skip expired bundles and clean up the set
                if active_bundle.expires_at <= chrono::Utc::now() {
                    let set_key = format!("bundle:active:set:{}", imsi);
                    let _: () = redis::AsyncCommands::srem(&mut conn, &set_key, &bundle_id)
                        .await
                        .unwrap_or(());
                    continue;
                }

                let has_allowance = match usage_type {
                    UsageType::Data  => active_bundle.remaining_allowances.data_bytes
                        .map_or(false, |b| b >= amount),
                    UsageType::Voice => active_bundle.remaining_allowances.voice_seconds
                        .map_or(false, |s| s >= amount),
                    UsageType::SMS   => active_bundle.remaining_allowances.sms_count
                        .map_or(false, |c| c >= amount),
                };

                if !has_allowance {
                    continue;
                }

                match usage_type {
                    UsageType::Data  => {
                        if let Some(ref mut b) = active_bundle.remaining_allowances.data_bytes { *b -= amount; }
                    }
                    UsageType::Voice => {
                        if let Some(ref mut s) = active_bundle.remaining_allowances.voice_seconds { *s -= amount; }
                    }
                    UsageType::SMS   => {
                        if let Some(ref mut c) = active_bundle.remaining_allowances.sms_count { *c -= amount; }
                    }
                }

                let active_key = format!("bundle:active:{}:{}", imsi, bundle_id);
                let updated_json = serde_json::to_string(&active_bundle)
                    .map_err(|e| ChargingError::SerializationError(e.to_string()))?;
                let _: () = redis::AsyncCommands::set(&mut conn, &active_key, updated_json)
                    .await
                    .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

                return Ok(true);
            }

            Ok(false)
        }).await.map_err(|e| match e {
            crate::circuit_breaker::CircuitBreakerError::Open =>
                ChargingError::RedisConnection("Circuit breaker is open".to_string()),
            crate::circuit_breaker::CircuitBreakerError::Inner(e) => e,
        })
    }


    pub async fn list_bundles(&self) -> ChargingResult<Vec<Bundle>> {
        use sqlx::Row;

        let rows = sqlx::query(
            "SELECT bundle_id, name, bundle_type, data_bytes, voice_seconds, \
             sms_count, roaming_data_bytes, validity_days, priority \
             FROM bundles WHERE is_active = TRUE ORDER BY priority ASC",
        )
        .fetch_all(&self.plans.pool)
        .await
        .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

        let bundles = rows
            .into_iter()
            .map(|row| Bundle {
                bundle_id: row.get("bundle_id"),
                name:      row.get("name"),
                bundle_type: BundleType::from_str(row.get("bundle_type")),
                allowances: BundleAllowances {
                    data_bytes:         row.get::<Option<i64>, _>("data_bytes").map(|v| v as u64),
                    voice_seconds:      row.get::<Option<i64>, _>("voice_seconds").map(|v| v as u64),
                    sms_count:          row.get::<Option<i64>, _>("sms_count").map(|v| v as u64),
                    roaming_data_bytes: row.get::<Option<i64>, _>("roaming_data_bytes").map(|v| v as u64),
                },
                validity_days: row.get::<i32, _>("validity_days") as u32,
                priority:      row.get::<i16, _>("priority") as u8,
                amount_unconverted: row.get::<i64, _>("amount_unconverted"),
                is_active:     true,
            })
            .collect();

        Ok(bundles)
    }

    pub async fn get_bundle_by_id(&self, bundle_id: &str) -> ChargingResult<Option<Bundle>> {
        match self.get_bundle_config(bundle_id).await {
            Ok(bundle) => Ok(Some(bundle)),
            Err(ChargingError::BundleNotFound(_)) => Ok(None),
            Err(e) => Err(e),
        }
    }


    pub async fn purchase_bundle_with_airtime(
        &self,
        imsi: &str,
        bundle_id: &str,
    ) -> ChargingResult<ActiveBundle> {
        use sqlx::Row;

        // 1. Load bundle definition from PostgreSQL
        let bundle = self.get_bundle_config(bundle_id).await?;

        if bundle.airtime_cost == 0 {
            return Err(ChargingError::InvalidBundleConfig(
                "Bundle has no airtime price set".to_string(),
            ));
        }

        // 2. Check and deduct airtime atomically in Redis (Lua script)
        let lua_script = r#"
            local key = KEYS[1]
            local cost = tonumber(ARGV[1])
            local balance = tonumber(redis.call('GET', key) or '0')
            if balance < cost then
                return {-1, balance}
            end
            local new_balance = redis.call('DECRBY', key, cost)
            return {0, new_balance}
        "#;

        let mut conn = self.redis_client
            .get_multiplexed_async_connection()
            .await
            .map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

        let airtime_key = format!("airtime:home:{}", imsi);
        let result: Vec<i64> = redis::Script::new(lua_script)
            .key(&airtime_key)
            .arg(bundle.airtime_cost as i64)
            .invoke_async(&mut conn)
            .await
            .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

        if result[0] == -1 {
            return Err(ChargingError::InsufficientAirtime {
                available: result[1] as i64,
                required:  bundle.airtime_cost,
            });
        }

        let new_airtime_balance = result[1] as u64;

        // 3. Activate the bundle in Redis (same as activate_bundle_for_subscriber)
        let active_bundle = self.activate_bundle_for_subscriber(imsi, bundle_id).await?;

        // 4. Persist airtime deduction to PostgreSQL (audit trail)
        self.postgres_circuit_breaker.execute(|| async {
            // Get subscriber_id for FK
            let sub_row = sqlx::query(
                "SELECT id FROM subscribers WHERE imsi = $1"
            )
            .bind(imsi)
            .fetch_optional(&self.plans.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?
            .ok_or_else(|| ChargingError::SubscriberNotFound(imsi.to_string()))?;

            let subscriber_id: i32 = sub_row.get("id");

            // Write airtime_transactions row
            sqlx::query(r#"
                INSERT INTO airtime_transactions
                    (subscriber_id, imsi, transaction_type, seconds_delta, balance_after, roaming, reference_id)
                VALUES ($1, $2, 'bundle_purchase', $3, $4, FALSE, $5)
            "#)
            .bind(subscriber_id)
            .bind(imsi)
            .bind(-(bundle.airtime_cost as i64))   // negative = debit
            .bind(new_airtime_balance as i64)
            .bind(bundle_id)
            .execute(&self.plans.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

            // Update airtime_balances table (PG source of truth)
            sqlx::query(r#"
                INSERT INTO airtime_balances (subscriber_id, imsi, home_seconds, updated_at)
                VALUES ($1, $2, $3, NOW())
                ON CONFLICT (imsi) DO UPDATE SET
                    home_seconds = $3,
                    updated_at   = NOW()
            "#)
            .bind(subscriber_id)
            .bind(imsi)
            .bind(new_airtime_balance as i64)
            .execute(&self.plans.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;

            Ok(())
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::DatabaseError("Circuit breaker open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })?;

        Ok(active_bundle)
    }

    pub async fn deactivate_bundle(&self, bundle_id: &str) -> ChargingResult<()> {
        sqlx::query("UPDATE bundles SET is_active = FALSE, updated_at = NOW() WHERE bundle_id = $1")
            .bind(bundle_id)
            .execute(&self.plans.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;
        Ok(())
    }

    async fn get_bundle_config(&self, bundle_id: &str) -> ChargingResult<Bundle> {
       let row = sqlx::query(r#"
           SELECT bundle_id, name, bundle_type, data_bytes, voice_seconds,
                  sms_count, roaming_data_bytes, validity_days, priority, amount_unconverted
           FROM bundles WHERE bundle_id = $1 AND is_active = TRUE
       "#)
        .bind(bundle_id)
        .fetch_optional(&self.plans.pool)
        .await
        .map_err(|e| ChargingError::DatabaseError(e.to_string()))?
        .ok_or_else(|| ChargingError::BundleNotFound(bundle_id.to_string()))?;

        Ok(Bundle {
            bundle_id: row.get("bundle_id"),
            name: row.get("name"),
            bundle_type: match row.get::<&str, _>("bundle_type") {
                "voice"  => BundleType::Voice,
                "sms"    => BundleType::SMS,
                "hybrid" => BundleType::Hybrid,
                _        => BundleType::Data,
            },
            allowances: BundleAllowances {
                data_bytes:         row.get::<Option<i64>, _>("data_bytes").map(|v| v as u64),
                voice_seconds:      row.get::<Option<i64>, _>("voice_seconds").map(|v| v as u64),
                sms_count:          row.get::<Option<i64>, _>("sms_count").map(|v| v as u64),
                roaming_data_bytes: row.get::<Option<i64>, _>("roaming_data_bytes").map(|v| v as u64),
            },
            validity_days: row.get::<i32, _>("validity_days") as u32,
            priority:      row.get::<i16, _>("priority") as u8,
            amount_unconverted: row.get::<i64, _>("amount_unconverted"),
            is_active:     true,
        })
    }

    async fn record_bundle_activation(
        &self,
        conn: &mut redis::aio::MultiplexedConnection,
        imsi: &str,
        bundle_id: &str,
    ) -> ChargingResult<()> {
        let audit_key = format!("bundle:audit:{}:{}", imsi, bundle_id);
        let payload = serde_json::json!({
            "imsi": imsi,
            "bundle_id": bundle_id,
            "activated_at": chrono::Utc::now().to_rfc3339(),
        }).to_string();
        let _: () = redis::AsyncCommands::set(conn, &audit_key, payload)
            .await
            .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;
        Ok(())
    }


    /// Resolve IMSI from MSISDN via PostgreSQL.
    async fn resolve_imsi_from_msisdn(&self, msisdn: &str) -> ChargingResult<String> {
        use sqlx::Row;
        let row = sqlx::query(
            "SELECT imsi FROM subscribers WHERE msisdn = $1 AND deleted_at IS NULL LIMIT 1",
        )
        .bind(msisdn)
        .fetch_optional(&self.plans.pool)
        .await
        .map_err(|e| ChargingError::DatabaseError(e.to_string()))?
        .ok_or_else(|| ChargingError::SubscriberNotFound(msisdn.to_string()))?;

        Ok(row.get("imsi"))
    }

    async fn deduct_airtime_atomic(
        &self,
        conn: &mut redis::aio::MultiplexedConnection,
        imsi: &str,
        amount: i64,
    ) -> ChargingResult<i64> {
        // Lua script: atomic check-and-deduct
        let lua_script = r#"
            local key     = KEYS[1]
            local cost    = tonumber(ARGV[1])
            local balance = tonumber(redis.call('GET', key) or '0')
            if balance < cost then
                return -1
            end
            return redis.call('DECRBY', key, cost)
        "#;

        let new_balance: i64 = redis::Script::new(lua_script)
            .key(format!("airtime:home:{}", imsi))
            .arg(amount)
            .invoke_async(conn)
            .await
            .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

        if new_balance < 0 {
            // Script returned -1 sentinel
            let current: i64 = redis::AsyncCommands::get(conn, format!("airtime:home:{}", imsi))
                .await
                .unwrap_or(0);
            return Err(ChargingError::InsufficientAirtime {
                available: current,
                required: amount,
            });
        }

        Ok(new_balance)
    }



    pub async fn purchase_bundle_with_airtime(
        &self,
        msisdn: &str,
        bundle_id: &str,
    ) -> ChargingResult<ActiveBundle> {
        // 1. Resolve IMSI
        let imsi = self.resolve_imsi_from_msisdn(msisdn).await?;

        // 2. Load bundle config
        let bundle = self.get_bundle_config(bundle_id).await?;

        // 3. Deduct airtime atomically
        let mut conn = self.redis_client.get_multiplexed_async_connection()
            .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

        let new_airtime = self.deduct_airtime_atomic(&mut conn, &imsi, bundle.amount_unconverted).await?;

        // 4. Activate bundle for subscriber
        let active_bundle = self.activate_bundle_for_subscriber(&imsi, bundle_id).await?;

        // 5. Audit: record airtime transaction
        self.record_airtime_transaction(
            &imsi,
            -bundle.amount_unconverted,
            new_airtime,
            &format!("bundle_purchase:{}", bundle_id),
        ).await?;

        Ok(active_bundle)
        }


        /// POST /v1/bundles/:id/gift
        /// Sender purchases a bundle for a recipient using the sender's airtime.
        /// Deducts from sender_msisdn's airtime; activates bundle on recipient_msisdn.
        pub async fn gift_bundle_with_airtime(
            &self,
            sender_msisdn: &str,
            recipient_msisdn: &str,
            bundle_id: &str,
        ) -> ChargingResult<ActiveBundle> {
            // 1. Resolve both IMSIs
            let sender_imsi    = self.resolve_imsi_from_msisdn(sender_msisdn).await?;
            let recipient_imsi = self.resolve_imsi_from_msisdn(recipient_msisdn).await?;

            // 2. Load bundle config
            let bundle = self.get_bundle_config(bundle_id).await?;

            // 3. Deduct from SENDER's airtime
            let mut conn = self.redis_client.get_multiplexed_async_connection()
                .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

            let new_sender_airtime = self.deduct_airtime_atomic(
                &mut conn, &sender_imsi, bundle.amount_unconverted,
            ).await?;

            // 4. Activate bundle on RECIPIENT
            let active_bundle = self.activate_bundle_for_subscriber(&recipient_imsi, bundle_id).await?;

            // 5. Audit: record airtime transaction for sender
            self.record_airtime_transaction(
                &sender_imsi,
                -bundle.amount_unconverted,
                new_sender_airtime,
                &format!("bundle_gift:{}:to:{}", bundle_id, recipient_imsi),
            ).await?;

            Ok(active_bundle)
        }

        /// Internal: write an airtime_transactions row for audit.
        async fn record_airtime_transaction(
            &self,
            imsi: &str,
            delta: i64,
            balance_after: i64,
            reason: &str,
        ) -> ChargingResult<()> {
            sqlx::query(r#"
                INSERT INTO airtime_transactions
                    (imsi, delta_seconds, balance_after, reason, created_at)
                VALUES ($1, $2, $3, $4, NOW())
            "#)
            .bind(imsi)
            .bind(delta)
            .bind(balance_after)
            .bind(reason)
            .execute(&self.plans.pool)
            .await
            .map_err(|e| ChargingError::DatabaseError(e.to_string()))?;
            Ok(())
        }
}
