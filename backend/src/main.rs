use std::path::Path;
use std::sync::Arc;

use axum::routing::{get, post};
use axum::Router;
use tokio::sync::broadcast;
use tower_http::cors::{Any, CorsLayer};
use tower_http::services::{ServeDir, ServeFile};

use std::collections::HashMap;

use server_monitor::api::{routes, websocket};
use server_monitor::config::{load_config, Settings};
use server_monitor::services::collector_manager::CollectorManager;
use server_monitor::services::metrics_store::MetricsStore;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "server_monitor=info".parse().unwrap()),
        )
        .init();

    let settings = Settings::from_env();
    tracing::info!(
        "Starting ServerMonitor on {}:{}",
        settings.host,
        settings.port
    );

    if settings.mock_mode {
        tracing::info!("Mock mode enabled");
    }

    // Load config
    let config_path = Path::new(&settings.config_path);
    let config = load_config(config_path);

    // Set up shared state
    let store = MetricsStore::new();
    let (tx, _rx) = broadcast::channel::<String>(64);

    let mut manager = CollectorManager::new(store.clone(), tx.clone(), settings.mock_mode);
    manager.load_config(&config);

    let collector_count = manager.collector_count();

    // Build system config map for action handlers
    let system_configs: HashMap<String, _> = config
        .systems
        .iter()
        .filter(|s| s.enabled)
        .map(|s| (s.id.clone(), s.config.clone()))
        .collect();

    let state = Arc::new(routes::AppState {
        store,
        manager: tokio::sync::Mutex::new(manager),
        settings: settings.clone(),
        broadcast_tx: tx,
        system_configs,
    });

    // Start collection loop
    {
        let mut mgr = state.manager.lock().await;
        mgr.start().await;
    }
    tracing::info!("Started monitoring {} systems", collector_count);

    // Build router
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    let static_dir = std::env::var("SERVERMONITOR_STATIC_DIR")
        .unwrap_or_else(|_| "/app/static".to_string());
    let index_file = format!("{}/index.html", static_dir);

    let app = Router::new()
        .route("/api/systems", get(routes::get_all_systems))
        .route("/api/systems/{system_id}", get(routes::get_system))
        .route("/api/health", get(routes::health_check))
        .route("/api/reload", post(routes::reload_config))
        .route(
            "/api/systems/{system_id}/actions/docker",
            post(routes::docker_action),
        )
        .route(
            "/api/systems/{system_id}/actions/qbittorrent",
            post(routes::qbit_action),
        )
        .route("/ws", get(websocket::ws_handler))
        .fallback_service(ServeDir::new(&static_dir).fallback(ServeFile::new(&index_file)))
        .layer(cors)
        .with_state(state.clone());

    // Start server
    let addr = format!("{}:{}", settings.host, settings.port);
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("Failed to bind address");

    tracing::info!("Listening on {}", addr);

    // Graceful shutdown
    let state_shutdown = state.clone();
    let shutdown_signal = async move {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to install CTRL+C handler");
        tracing::info!("Shutting down...");
        let mut mgr = state_shutdown.manager.lock().await;
        mgr.stop().await;
        tracing::info!("Stopped monitoring");
    };

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal)
        .await
        .expect("Server error");
}
