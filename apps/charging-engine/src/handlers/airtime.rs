use axum::{extract::{Path, State}, http::StatusCode, response::Json, Json as AxumJson};
use serde_json::json;
use crate::models::AppState;
use crate::errors::ChargingError;

#[derive(Deserialize)]
pub struct AddAirtimeRequest {
    pub seconds_to_add: u64,
    pub roaming: Option<bool>,
}

#[derive(Serialize)]
pub struct AirtimeBalanceResponse {
    pub imsi: String,
    pub home_seconds: u64,
    pub roaming_seconds: u64,
    pub last_updated: chrono::DateTime<chrono::Utc>,
}


#[derive(Deserialize)]
pub struct DeductAirtimeRequest {
    pub seconds_to_deduct: u64,
    pub roaming: Option<bool>,
}

pub async fn add_airtime(
    State(state): State<AppState>,
    Path(imsi): Path<String>,
    AxumJson(request): AxumJson<AddAirtimeRequest>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let roaming = request.roaming.unwrap_or(false);

    match state.charging_engine.add_airtime(&imsi, request.seconds_to_add, roaming).await {
        Ok(new_balance) => Ok(Json(json!({
            "imsi": imsi,
            "roaming": roaming,
            "new_balance_seconds": new_balance,
            "timestamp": chrono::Utc::now()
        }))),
        Err(ChargingError::InvalidInput(_)) => Err(StatusCode::BAD_REQUEST),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

pub async fn get_airtime_balance(
    State(state): State<AppState>,
    Path(imsi): Path<String>,
) -> Result<Json<AirtimeBalanceResponse>, StatusCode> {
    match state.charging_engine.get_airtime_balance(&imsi).await {
        Ok(balance) => Ok(Json(AirtimeBalanceResponse {
            imsi: balance.imsi,
            home_seconds: balance.home_seconds,
            roaming_seconds: balance.roaming_seconds,
            last_updated: balance.last_updated,
        })),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

pub async fn deduct_airtime(
    State(state): State<AppState>,
    Path(imsi): Path<String>,
    Json(request): Json<DeductAirtimeRequest>,
) -> ChargingResult<Json<serde_json::Value>> {
    let roaming = request.roaming.unwrap_or(false);
    let result = state.charging_engine
        .deduct_airtime(&imsi, request.seconds_to_deduct, roaming)
        .await
        .with_context("Failed to deduct airtime")?;
    Ok(Json(serde_json::json!({
        "imsi": imsi,
        "deducted_seconds": result.deducted,
        "new_balance_seconds": result.new_balance,
        "roaming": roaming,
    })))
}
