use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};

#[derive(Debug, thiserror::Error)]
pub enum CollectorError {
    #[error("SSH error: {0}")]
    Ssh(String),
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),
    #[error("Docker error: {0}")]
    Docker(#[from] bollard::errors::Error),
    #[error("Parse error: {0}")]
    Parse(String),
    #[error("Connection failed: {0}")]
    Connection(String),
    #[error("Config error: {0}")]
    Config(String),
}

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("Not found: {0}")]
    NotFound(String),
    #[error("Internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match &self {
            AppError::NotFound(msg) => (StatusCode::NOT_FOUND, msg.clone()),
            AppError::Internal(err) => {
                (StatusCode::INTERNAL_SERVER_ERROR, err.to_string())
            }
        };
        (status, serde_json::json!({"error": message}).to_string()).into_response()
    }
}
