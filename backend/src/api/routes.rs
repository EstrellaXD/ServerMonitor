use std::path::Path;
use std::sync::Arc;

use axum::extract::{Path as AxumPath, State};
use axum::response::Json;
use tokio::sync::broadcast;

use crate::config::{load_config, Settings};
use crate::services::collector_manager::CollectorManager;
use crate::services::metrics_store::MetricsStore;

pub struct AppState {
    pub store: MetricsStore,
    pub manager: tokio::sync::Mutex<CollectorManager>,
    pub settings: Settings,
    pub broadcast_tx: broadcast::Sender<String>,
}

pub async fn root() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "name": "ServerMonitor",
        "version": "1.0.0",
        "docs": "/docs",
    }))
}

pub async fn health_check() -> Json<serde_json::Value> {
    Json(serde_json::json!({"status": "ok"}))
}

pub async fn get_all_systems(
    State(state): State<Arc<AppState>>,
) -> Json<serde_json::Value> {
    let systems = state.store.get_all().await;
    Json(serde_json::to_value(systems).unwrap_or_default())
}

pub async fn get_system(
    State(state): State<Arc<AppState>>,
    AxumPath(system_id): AxumPath<String>,
) -> Json<serde_json::Value> {
    match state.store.get(&system_id).await {
        Some(metrics) => Json(serde_json::to_value(metrics).unwrap_or_default()),
        None => Json(serde_json::json!({"error": "System not found"})),
    }
}

pub async fn reload_config(
    State(state): State<Arc<AppState>>,
) -> Json<serde_json::Value> {
    let config_path_str = &state.settings.config_path;
    let config_path = Path::new(config_path_str);

    let config = load_config(config_path);

    let mut manager = state.manager.lock().await;
    manager.stop().await;
    state.store.clear().await;
    manager.load_config(&config);
    manager.start().await;

    let ids = manager.collector_ids();

    Json(serde_json::json!({
        "status": "success",
        "message": format!("Config reloaded successfully. Monitoring {} systems", ids.len()),
        "systems": ids,
    }))
}
