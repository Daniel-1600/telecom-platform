use crate::circuit_breaker::CircuitBreakerError;
use crate::errors::{ChargingError, ChargingResult};
use super::types::AirtimeBalance;

impl crate::charging::ChargingEngine {
    pub async fn add_airtime(&self, imsi: &str, seconds_to_add: u64, roaming: bool) -> ChargingResult<u64> {
        self.redis_circuit_breaker.execute(|| async {
            let mut conn = self.redis_client.get_multiplexed_async_connection()
                .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

            let key = if roaming {
                format!("airtime:roaming:{}", imsi)
            } else {
                format!("airtime:home:{}", imsi)
            };

            // Atomic increment with overflow check
            let current: Option<u64> = redis::AsyncCommands::get(&mut conn, &key).await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            if let Some(existing) = current {
                if existing.checked_add(seconds_to_add).is_none() {
                    return Err(ChargingError::InvalidInput("Airtime addition would overflow".to_string()));
                }
            }

            let new_balance: u64 = redis::AsyncCommands::incr(&mut conn, &key, seconds_to_add).await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            // Set expiration (30 days for airtime)
            let _: () = redis::AsyncCommands::expire(&mut conn, &key, 2592000).await
                .unwrap_or(());

            // Update metadata
            let meta_key = format!("airtime:meta:{}", imsi);
            let _: () = redis::AsyncCommands::hset(&mut conn, &meta_key, "last_updated", Utc::now().timestamp()).await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            Ok(new_balance)
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::RedisConnection("Circuit breaker is open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }

    pub async fn deduct_airtime(&self, imsi: &str, seconds_to_deduct: u64, roaming: bool) -> ChargingResult<AirtimeDeduction> {
        self.redis_circuit_breaker.execute(|| async {
            let mut conn = self.redis_client.get_multiplexed_async_connection()
                .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

            let balance_key = if roaming {
                format!("airtime:roaming:{}", imsi)
            } else {
                format!("airtime:home:{}", imsi)
            };

            // Use Lua script for atomic check-and-deduct
            let script = r#"
                local current = redis.call('GET', KEYS[1])
                if not current then
                    return {err = "NO_BALANCE"}
                end
                current = tonumber(current)
                if current < tonumber(ARGV[1]) then
                    return {err = "INSUFFICIENT_BALANCE", current = current}
                end
                local new_balance = current - tonumber(ARGV[1])
                redis.call('SET', KEYS[1], new_balance)
                return {ok = true, new_balance = new_balance, old_balance = current}
            "#;

            let result: redis::RedisResult<redis::Value> = redis::Script::new(script)
                .key(&balance_key)
                .arg(seconds_to_deduct)
                .invoke_async(&mut conn)
                .await;

            match result {
                Ok(redis::Value::Bulk(items)) => {
                    let deduction = self.parse_airtime_deduction_result(items)?;

                    // Record usage event for billing
                    self.record_airtime_usage(&mut conn, imsi, seconds_to_deduct, roaming).await?;

                    Ok(deduction)
                }
                Ok(_) => Err(ChargingError::InvalidState("Unexpected script result".to_string())),
                Err(e) => Err(ChargingError::RedisOperation(e.to_string())),
            }
        }).await
    }

    pub async fn get_airtime_balance(&self, imsi: &str) -> ChargingResult<AirtimeBalance> {
        self.redis_circuit_breaker.execute(|| async {
            let mut conn = self.redis_client.get_multiplexed_async_connection()
                .await.map_err(|e| ChargingError::RedisConnection(e.to_string()))?;

            let home_key = format!("airtime:home:{}", imsi);
            let roaming_key = format!("airtime:roaming:{}", imsi);
            let meta_key = format!("airtime:meta:{}", imsi);

            let pipe = redis::pipe();
            pipe.get(&home_key)
                .get(&roaming_key)
                .hget(&meta_key, "last_updated");

            let results: Vec<Option<redis::Value>> = pipe.query_async(&mut conn).await
                .map_err(|e| ChargingError::RedisOperation(e.to_string()))?;

            let home_seconds = match results[0] {
                Some(redis::Value::Int(v)) => v as u64,
                _ => 0,
            };

            let roaming_seconds = match results[1] {
                Some(redis::Value::Int(v)) => v as u64,
                _ => 0,
            };

            let last_updated = match results[2] {
                Some(redis::Value::Bulk(v)) => {
                    if let Some(redis::Value::Data(ts_bytes)) = v.first() {
                        let ts_str = String::from_utf8_lossy(ts_bytes);
                        ts_str.parse::<i64>().ok()
                            .and_then(|ts| DateTime::from_timestamp(ts, 0))
                            .unwrap_or_else(Utc::now)
                    } else {
                        Utc::now()
                    }
                }
                _ => Utc::now(),
            };

            Ok(AirtimeBalance {
                imsi: imsi.to_string(),
                home_seconds,
                roaming_seconds,
                last_updated,
            })
        }).await.map_err(|e| match e {
            CircuitBreakerError::Open => ChargingError::RedisConnection("Circuit breaker is open".to_string()),
            CircuitBreakerError::Inner(e) => e,
        })
    }
}

#[derive(Debug, Serialize)]
pub struct AirtimeDeduction {
    pub old_balance: u64,
    pub new_balance: u64,
    pub deducted: u64,
    pub remaining: u64,
}
