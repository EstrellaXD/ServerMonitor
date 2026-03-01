use std::collections::HashMap;
use std::path::Path;
use std::sync::Arc;

use axum::extract::{Path as AxumPath, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Json, Response};
use bollard::Docker;
use reqwest::Client;
use serde::Deserialize;
use tokio::sync::broadcast;

use crate::config::{
    load_config, DockerSystemConfig, QBittorrentSystemConfig, Settings, SystemConfigType,
};
use crate::services::collector_manager::CollectorManager;
use crate::services::metrics_store::MetricsStore;

pub struct AppState {
    pub store: MetricsStore,
    pub manager: tokio::sync::Mutex<CollectorManager>,
    pub settings: Settings,
    pub broadcast_tx: broadcast::Sender<String>,
    pub system_configs: HashMap<String, SystemConfigType>,
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

// --- Action request types ---

#[derive(Deserialize)]
pub struct DockerActionRequest {
    pub container_name: String,
    pub action: String,
}

#[derive(Deserialize)]
pub struct QbitActionRequest {
    pub hash: String,
    pub action: String,
    #[serde(default)]
    pub delete_files: bool,
}

// --- Action error response helper ---

fn action_error(status: StatusCode, msg: &str) -> Response {
    (status, Json(serde_json::json!({"error": msg}))).into_response()
}

// --- Docker action handler ---

fn connect_docker(config: &DockerSystemConfig) -> Result<Docker, String> {
    if let Some(ref host) = config.host {
        Docker::connect_with_http(host, 10, bollard::API_DEFAULT_VERSION)
            .map_err(|e| format!("Docker connect failed: {e}"))
    } else {
        Docker::connect_with_socket(&config.socket, 10, bollard::API_DEFAULT_VERSION)
            .map_err(|e| format!("Docker socket connect failed: {e}"))
    }
}

pub async fn docker_action(
    State(state): State<Arc<AppState>>,
    AxumPath(system_id): AxumPath<String>,
    Json(body): Json<DockerActionRequest>,
) -> Response {
    // Look up Docker config
    let config = match state.system_configs.get(&system_id) {
        Some(SystemConfigType::Docker(c)) => c.clone(),
        _ => return action_error(StatusCode::NOT_FOUND, "Docker system not found"),
    };

    // Validate action
    if !matches!(body.action.as_str(), "start" | "stop" | "restart") {
        return action_error(
            StatusCode::BAD_REQUEST,
            &format!("Invalid action: {}", body.action),
        );
    }

    // Connect
    let client = match connect_docker(&config) {
        Ok(c) => c,
        Err(e) => return action_error(StatusCode::INTERNAL_SERVER_ERROR, &e),
    };

    // Execute action
    let result = match body.action.as_str() {
        "start" => client
            .start_container::<String>(&body.container_name, None)
            .await
            .map(|_| ()),
        "stop" => client
            .stop_container(&body.container_name, None)
            .await
            .map(|_| ()),
        "restart" => client
            .restart_container(&body.container_name, None)
            .await
            .map(|_| ()),
        _ => unreachable!(),
    };

    match result {
        Ok(()) => {
            tracing::info!(
                "[{}] Docker action '{}' on container '{}'",
                system_id,
                body.action,
                body.container_name
            );
            (
                StatusCode::OK,
                Json(serde_json::json!({
                    "status": "success",
                    "action": body.action,
                    "container": body.container_name,
                })),
            )
                .into_response()
        }
        Err(e) => {
            tracing::error!(
                "[{}] Docker action '{}' failed on '{}': {}",
                system_id,
                body.action,
                body.container_name,
                e
            );
            action_error(StatusCode::INTERNAL_SERVER_ERROR, &e.to_string())
        }
    }
}

// --- qBittorrent action handler ---

async fn qbit_authenticate(client: &Client, config: &QBittorrentSystemConfig) -> Result<(), String> {
    let resp = client
        .post(format!("{}/api/v2/auth/login", config.url))
        .form(&[
            ("username", config.username.as_str()),
            ("password", config.password.as_str()),
        ])
        .send()
        .await
        .map_err(|e| format!("qBittorrent login request failed: {e}"))?;

    let text = resp.text().await.map_err(|e| e.to_string())?;
    if text != "Ok." {
        return Err("qBittorrent authentication failed".to_string());
    }
    Ok(())
}

pub async fn qbit_action(
    State(state): State<Arc<AppState>>,
    AxumPath(system_id): AxumPath<String>,
    Json(body): Json<QbitActionRequest>,
) -> Response {
    // Look up qBit config
    let config = match state.system_configs.get(&system_id) {
        Some(SystemConfigType::QBittorrent(c)) => c.clone(),
        _ => return action_error(StatusCode::NOT_FOUND, "qBittorrent system not found"),
    };

    // Validate action
    if !matches!(body.action.as_str(), "resume" | "pause" | "delete") {
        return action_error(
            StatusCode::BAD_REQUEST,
            &format!("Invalid action: {}", body.action),
        );
    }

    // Create client with cookie store for session
    let client = match Client::builder()
        .danger_accept_invalid_certs(true)
        .cookie_store(true)
        .timeout(std::time::Duration::from_secs(10))
        .build()
    {
        Ok(c) => c,
        Err(e) => return action_error(StatusCode::INTERNAL_SERVER_ERROR, &e.to_string()),
    };

    // Authenticate
    if let Err(e) = qbit_authenticate(&client, &config).await {
        return action_error(StatusCode::INTERNAL_SERVER_ERROR, &e);
    }

    // Build the qBittorrent API request
    let (endpoint, form_data) = match body.action.as_str() {
        "resume" => (
            format!("{}/api/v2/torrents/resume", config.url),
            vec![("hashes".to_string(), body.hash.clone())],
        ),
        "pause" => (
            format!("{}/api/v2/torrents/pause", config.url),
            vec![("hashes".to_string(), body.hash.clone())],
        ),
        "delete" => (
            format!("{}/api/v2/torrents/delete", config.url),
            vec![
                ("hashes".to_string(), body.hash.clone()),
                ("deleteFiles".to_string(), body.delete_files.to_string()),
            ],
        ),
        _ => unreachable!(),
    };

    // Execute
    let result = client.post(&endpoint).form(&form_data).send().await;

    match result {
        Ok(resp) if resp.status().is_success() => {
            tracing::info!(
                "[{}] qBit action '{}' on torrent '{}'",
                system_id,
                body.action,
                body.hash
            );
            (
                StatusCode::OK,
                Json(serde_json::json!({
                    "status": "success",
                    "action": body.action,
                    "hash": body.hash,
                })),
            )
                .into_response()
        }
        Ok(resp) => {
            let status = resp.status();
            let text = resp.text().await.unwrap_or_default();
            tracing::error!(
                "[{}] qBit action '{}' failed: {} {}",
                system_id,
                body.action,
                status,
                text
            );
            action_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                &format!("qBittorrent API returned {status}"),
            )
        }
        Err(e) => {
            tracing::error!(
                "[{}] qBit action '{}' request failed: {}",
                system_id,
                body.action,
                e
            );
            action_error(StatusCode::INTERNAL_SERVER_ERROR, &e.to_string())
        }
    }
}
