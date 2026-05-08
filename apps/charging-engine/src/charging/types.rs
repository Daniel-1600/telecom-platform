use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use redis::{FromRedisValue, ToRedisArgs, ToSingleRedisArg};

// ─── Charging Rule ────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum ChargingRule {
    Allowed,
    InsufficientCredit,
    DataLimitExceeded,
    VoiceLimitExceeded,
    SmsLimitExceeded,
    UserBlocked,
    Blocked,
}

impl ChargingRule {
    pub fn as_str(&self) -> &str {
        match self {
            ChargingRule::Allowed             => "ALLOWED",
            ChargingRule::InsufficientCredit  => "INSUFFICIENT_CREDIT",
            ChargingRule::DataLimitExceeded   => "DATA_LIMIT_EXCEEDED",
            ChargingRule::VoiceLimitExceeded  => "VOICE_LIMIT_EXCEEDED",
            ChargingRule::SmsLimitExceeded    => "SMS_LIMIT_EXCEEDED",
            ChargingRule::UserBlocked         => "USER_BLOCKED",
            ChargingRule::Blocked             => "BLOCKED",
        }
    }
}

// ─── Account ──────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AccountStatus {
    Active,
    Suspended,
    Terminated,
    Blocked,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubscriberAccount {
    pub imsi:         String,
    pub balance:      i64,
    pub data_limit:   u64,
    pub data_used:    u64,
    pub voice_limit:  u64,
    pub voice_used:   u64,
    pub sms_limit:    u64,
    pub sms_used:     u64,
    pub status:       AccountStatus,
    pub last_updated: DateTime<Utc>,
}

// ─── Usage ────────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum UsageType {
    Data,
    Voice,
    SMS,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UsageEvent {
    pub imsi:       String,
    pub session_id: String,
    pub usage_type: UsageType,
    pub volume:     u64,
    pub timestamp:  DateTime<Utc>,
    pub rate:       f64,
    pub cost:       f64,
}

// ─── Rating Plan ──────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RatingPlan {
    pub plan_id:     String,
    pub name:        String,
    pub data_rate:   f64,
    pub voice_rate:  f64,
    pub sms_rate:    f64,
    pub monthly_fee: f64,
    pub data_limit:  u64,
    pub voice_limit: u64,
    pub sms_limit:   u64,
}

// ─── Airtime ──────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AirtimeBalance {
    pub imsi:            String,
    pub home_seconds:    u64,
    pub roaming_seconds: u64,
    pub last_updated:    DateTime<Utc>,
}

/// Returned by `deduct_airtime` so callers know exactly what was consumed.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AirtimeDeduction {
    pub imsi:        String,
    pub deducted:    u64,
    pub new_balance: u64,
    pub roaming:     bool,
}

// ─── Bundles ──────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BundleType {
    Data,
    Voice,
    SMS,
    Hybrid,
}

impl BundleType {
    /// Returns the lowercase string stored in the `bundles.bundle_type` column.
    pub fn as_str(&self) -> &str {
        match self {
            BundleType::Data   => "data",
            BundleType::Voice  => "voice",
            BundleType::SMS    => "sms",
            BundleType::Hybrid => "hybrid",
        }
    }

    /// Parse from the DB string back to the enum.
    pub fn from_str(s: &str) -> Self {
        match s {
            "voice"  => BundleType::Voice,
            "sms"    => BundleType::SMS,
            "hybrid" => BundleType::Hybrid,
            _        => BundleType::Data,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BundleAllowances {
    pub data_bytes:         Option<u64>,
    pub voice_seconds:      Option<u64>,
    pub sms_count:          Option<u64>,
    pub roaming_data_bytes: Option<u64>,
}

/// Bundle definition stored in PostgreSQL.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Bundle {
    pub bundle_id:          String,
    pub name:               String,
    pub bundle_type:        BundleType,
    pub allowances:         BundleAllowances,
    pub validity_days:      u32,
    pub priority:           u8,
    pub amount_unconverted: i64,
    pub is_active:          bool,
}

/// A bundle that has been activated for a specific subscriber.
/// Stored as JSON in Redis with a TTL matching `validity_days`.
/// `priority` is copied from the bundle definition so `consume_from_bundle`
/// can sort without an extra DB round-trip.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActiveBundle {
    pub imsi:                 String,
    pub bundle_id:            String,
    /// Copied from `Bundle::priority` at activation time.
    pub priority:             u8,
    pub activated_at:         DateTime<Utc>,
    pub expires_at:           DateTime<Utc>,
    pub remaining_allowances: BundleAllowances,
}

// ─── Redis trait implementations ──────────────────────────────────────────────

impl FromRedisValue for SubscriberAccount {
    fn from_redis_value(v: redis::Value) -> Result<Self, redis::ParsingError> {
        let json: String = redis::from_redis_value(v)?;
        serde_json::from_str(&json).map_err(|e| redis::ParsingError::from(e.to_string()))
    }
}

impl ToRedisArgs for SubscriberAccount {
    fn write_redis_args<W>(&self, out: &mut W)
    where W: redis::RedisWrite + ?Sized {
        let json = serde_json::to_string(self).unwrap_or_else(|_| {
            r#"{"imsi":"","balance":0,"data_limit":0,"data_used":0,"voice_limit":0,"voice_used":0,"sms_limit":0,"sms_used":0,"status":"Active","last_updated":"1970-01-01T00:00:00Z"}"#.to_string()
        });
        json.write_redis_args(out)
    }
}

impl ToSingleRedisArg for SubscriberAccount {}

impl FromRedisValue for UsageEvent {
    fn from_redis_value(v: redis::Value) -> Result<Self, redis::ParsingError> {
        let json: String = redis::from_redis_value(v)?;
        serde_json::from_str(&json).map_err(|e| redis::ParsingError::from(e.to_string()))
    }
}

impl ToRedisArgs for UsageEvent {
    fn write_redis_args<W>(&self, out: &mut W)
    where W: redis::RedisWrite + ?Sized {
        let json = serde_json::to_string(self).unwrap_or_else(|_| {
            r#"{"imsi":"","session_id":"","usage_type":"Data","volume":0,"timestamp":"1970-01-01T00:00:00Z","rate":0.0,"cost":0.0}"#.to_string()
        });
        json.write_redis_args(out)
    }
}

impl ToSingleRedisArg for UsageEvent {}

/// ActiveBundle is stored as JSON in Redis — implement the Redis traits
/// so it can be set/get with `redis::AsyncCommands`.
impl FromRedisValue for ActiveBundle {
    fn from_redis_value(v: redis::Value) -> Result<Self, redis::ParsingError> {
        let json: String = redis::from_redis_value(v)?;
        serde_json::from_str(&json).map_err(|e| redis::ParsingError::from(e.to_string()))
    }
}

impl ToRedisArgs for ActiveBundle {
    fn write_redis_args<W>(&self, out: &mut W)
    where W: redis::RedisWrite + ?Sized {
        let json = serde_json::to_string(self)
            .unwrap_or_else(|_| "{}".to_string());
        json.write_redis_args(out)
    }
}

impl ToSingleRedisArg for ActiveBundle {}
