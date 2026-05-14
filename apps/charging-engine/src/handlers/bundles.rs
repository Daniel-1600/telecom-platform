use axum::{extract::{Path, State}, http::StatusCode, response::Json, Json as AxumJson};
use serde_json::json;
use crate::models::AppState;
use crate::errors::ChargingError;

#[derive(Deserialize)]
pub struct CreateBundleRequest {
    pub bundle_id: String,
    pub name: String,
    pub bundle_type: String,
    pub allowances: BundleAllowancesRequest,
    pub validity_days: u32,
    pub priority: Option<u8>,
}

#[derive(Deserialize)]
pub struct BundleAllowancesRequest {
    pub data_bytes: Option<u64>,
    pub voice_seconds: Option<u64>,
    pub sms_count: Option<u64>,
    pub roaming_data_bytes: Option<u64>,
}

#[derive(Deserialize)]
pub struct ActivateBundleRequest {
    pub msisdn: String,
}

#[derive(Serialize)]
pub struct BundleResponse {
    pub bundle_id: String,
    pub name: String,
    pub bundle_type: String,
    pub allowances: BundleAllowancesRequest,
    pub validity_days: u32,
    pub priority: u8,
    pub is_active: bool,
}

#[derive(Serialize)]
pub struct ActiveBundleResponse {
    pub imsi: String,
    pub bundle_id: String,
    pub activated_at: chrono::DateTime<chrono::Utc>,
    pub expires_at: chrono::DateTime<chrono::Utc>,
    pub remaining_allowances: BundleAllowancesRequest,
}

#[derive(Deserialize)]
pub struct PurchaseBundleRequest {
    pub msisdn: String,
}

#[derive(Deserialize)]
pub struct GiftBundleRequest {
    pub sender_msisdn:    String,
    pub recipient_msisdn: String,
}

/// create bunndle
pub async fn create_bundle(
    State(state): State<AppState>,
    AxumJson(request): AxumJson<CreateBundleRequest>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let bundle_type = match request.bundle_type.as_str() {
        "data" => crate::charging::types::BundleType::Data,
        "voice" => crate::charging::types::BundleType::Voice,
        "sms" => crate::charging::types::BundleType::SMS,
        "hybrid" => crate::charging::types::BundleType::Hybrid,
        _ => return Err(StatusCode::BAD_REQUEST),
    };

    let bundle = crate::charging::types::Bundle {
        bundle_id: request.bundle_id.clone(),
        name: request.name,
        bundle_type,
        allowances: crate::charging::types::BundleAllowances {
            data_bytes: request.allowances.data_bytes,
            voice_seconds: request.allowances.voice_seconds,
            sms_count: request.allowances.sms_count,
            roaming_data_bytes: request.allowances.roaming_data_bytes,
        },
        validity_days: request.validity_days,
        priority: request.priority.unwrap_or(1),
        amount_unconverted: request.amount_unconverted,
        is_active: true,
    };

    match state.charging_engine.create_bundle(bundle).await {
        Ok(_) => Ok(Json(json!({
            "status": "created",
            "bundle_id": request.bundle_id,
        }))),
        Err(ChargingError::InvalidInput(_)) => Err(StatusCode::BAD_REQUEST),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

/// Activate a bundle for a subscriber
pub async fn activate_bundle(
    State(state): State<AppState>,
    Path(bundle_id): Path<String>,
    AxumJson(request): AxumJson<ActivateBundleRequest>,
) -> Result<Json<ActiveBundleResponse>, StatusCode> {
    match state.charging_engine.activate_bundle_for_subscriber(&request.imsi, &bundle_id).await {
        Ok(active_bundle) => Ok(Json(ActiveBundleResponse {
            imsi: active_bundle.imsi,
            bundle_id: active_bundle.bundle_id,
            activated_at: active_bundle.activated_at,
            expires_at: active_bundle.expires_at,
            remaining_allowances: BundleAllowancesRequest {
                data_bytes: active_bundle.remaining_allowances.data_bytes,
                voice_seconds: active_bundle.remaining_allowances.voice_seconds,
                sms_count: active_bundle.remaining_allowances.sms_count,
                roaming_data_bytes: active_bundle.remaining_allowances.roaming_data_bytes,
            },
        })),
        Err(ChargingError::BundleNotFound(_)) => Err(StatusCode::NOT_FOUND),
        Err(ChargingError::BundleAlreadyActive) => Err(StatusCode::CONFLICT),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

/// List active bundles for a subscriber
pub async fn list_active_bundles(
    State(state): State<AppState>,
    Path(imsi): Path<String>,
) -> Result<Json<Vec<ActiveBundleResponse>>, StatusCode> {
    match state.charging_engine.get_active_bundles_for_subscriber(&imsi).await {
        Ok(bundles) => {
            let response: Vec<ActiveBundleResponse> = bundles.into_iter().map(|b| ActiveBundleResponse {
                imsi: b.imsi,
                bundle_id: b.bundle_id,
                activated_at: b.activated_at,
                expires_at: b.expires_at,
                remaining_allowances: BundleAllowancesRequest {
                    data_bytes: b.remaining_allowances.data_bytes,
                    voice_seconds: b.remaining_allowances.voice_seconds,
                    sms_count: b.remaining_allowances.sms_count,
                    roaming_data_bytes: b.remaining_allowances.roaming_data_bytes,
                },
            }).collect();
            Ok(Json(response))
        },
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

/// Consume from bundle allowances (internal use)
pub async fn consume_from_bundle(
    State(state): State<AppState>,
    AxumJson(request): AxumJson<serde_json::Value>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let imsi = request.get("imsi").and_then(|v| v.as_str()).ok_or(StatusCode::BAD_REQUEST)?;
    let usage_type_str = request.get("usage_type").and_then(|v| v.as_str()).ok_or(StatusCode::BAD_REQUEST)?;
    let amount = request.get("amount").and_then(|v| v.as_u64()).ok_or(StatusCode::BAD_REQUEST)?;

    let usage_type = match usage_type_str {
        "data" => crate::charging::types::UsageType::Data,
        "voice" => crate::charging::types::UsageType::Voice,
        "sms" => crate::charging::types::UsageType::SMS,
        _ => return Err(StatusCode::BAD_REQUEST),
    };

    match state.charging_engine.consume_from_bundle(imsi, usage_type, amount).await {
        Ok(consumed) => Ok(Json(json!({
            "consumed": consumed,
            "imsi": imsi,
            "usage_type": usage_type_str,
            "amount": amount,
        }))),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }



}



/// GET /v1/bundles
pub async fn list_bundles(
    State(state): State<AppState>,
) -> ChargingResult<Json<serde_json::Value>> {
    let bundles = state.charging_engine.list_bundles().await
        .with_context("Failed to list bundles")?;
    Ok(Json(serde_json::json!({ "bundles": bundles })))
}

/// GET /v1/bundles/:id
pub async fn get_bundle(
    Path(bundle_id): Path<String>,
    State(state): State<AppState>,
) -> ChargingResult<Json<serde_json::Value>> {
    let bundle = state.charging_engine.get_bundle_by_id(&bundle_id).await
        .with_context("Failed to get bundle")?;
    match bundle {
        Some(b) => Ok(Json(serde_json::json!({
            "bundle_id": b.bundle_id,
            "name": b.name,
            "validity_days": b.validity_days,
            "priority": b.priority,
            "is_active": b.is_active,
        }))),
        None => Err(ChargingError::BundleNotFound(bundle_id)),
    }
}

/// DELETE /v1/bundles/:id
pub async fn deactivate_bundle(
    Path(bundle_id): Path<String>,
    State(state): State<AppState>,
) -> ChargingResult<Json<serde_json::Value>> {
    state.charging_engine.deactivate_bundle(&bundle_id).await
        .with_context("Failed to deactivate bundle")?;
    Ok(Json(serde_json::json!({ "status": "deactivated", "bundle_id": bundle_id })))
}


/// POST /v1/bundles/:id/purchase
/// Purchase a bundle by spending airtime seconds.
pub async fn purchase_bundle_with_airtime(
    State(state): State<AppState>,
    Path(bundle_id): Path<String>,
    Json(req): Json<PurchaseBundleRequest>,
) -> ChargingResult<Json<serde_json::Value>> {
    let active = state.charging_engine
        .purchase_bundle_with_airtime(&req.imsi, &bundle_id)
        .await
        .with_context("Failed to purchase bundle with airtime")?;

    Ok(Json(serde_json::json!({
        "status": "purchased",
        "imsi": active.imsi,
        "bundle_id": active.bundle_id,
        "expires_at": active.expires_at,
        "remaining_allowances": active.remaining_allowances,
    })))
}


/// POST /v1/bundles/:id/purchase
pub async fn purchase_bundle(
    State(state): State<AppState>,
    Path(bundle_id): Path<String>,
    Json(req): Json<PurchaseBundleRequest>,
) -> ChargingResult<Json<serde_json::Value>> {
    let active = state.charging_engine
        .purchase_bundle_with_airtime(&req.msisdn, &bundle_id)
        .await
        .with_context("Failed to purchase bundle")?;

    Ok(Json(serde_json::json!({
        "status":               "purchased",
        "msisdn":               req.msisdn,
        "bundle_id":            active.bundle_id,
        "expires_at":           active.expires_at,
        "remaining_allowances": active.remaining_allowances,
    })))
}


pub async fn gift_bundle(
    State(state): State<AppState>,
    Path(bundle_id): Path<String>,
    Json(req): Json<GiftBundleRequest>,
) -> ChargingResult<Json<serde_json::Value>> {
    let active = state.charging_engine
        .gift_bundle_with_airtime(&req.sender_msisdn, &req.recipient_msisdn, &bundle_id)
        .await
        .with_context("Failed to gift bundle")?;

    Ok(Json(serde_json::json!({
        "status":               "gifted",
        "sender_msisdn":        req.sender_msisdn,
        "recipient_msisdn":     req.recipient_msisdn,
        "bundle_id":            active.bundle_id,
        "expires_at":           active.expires_at,
        "remaining_allowances": active.remaining_allowances,
    })))
}
