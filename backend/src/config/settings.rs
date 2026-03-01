use std::env;

#[derive(Debug, Clone)]
pub struct Settings {
    pub config_path: String,
    pub host: String,
    pub port: u16,
    pub debug: bool,
    pub mock_mode: bool,
}

impl Settings {
    pub fn from_env() -> Self {
        Self {
            config_path: env::var("SERVERMONITOR_CONFIG_PATH")
                .unwrap_or_else(|_| "config.yaml".to_string()),
            host: env::var("SERVERMONITOR_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: env::var("SERVERMONITOR_PORT")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(8742),
            debug: env::var("SERVERMONITOR_DEBUG")
                .map(|v| v == "true" || v == "1")
                .unwrap_or(false),
            mock_mode: env::var("SERVERMONITOR_MOCK_MODE")
                .map(|v| v == "true" || v == "1")
                .unwrap_or(false),
        }
    }
}
