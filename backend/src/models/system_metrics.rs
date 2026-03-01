use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use super::metrics::*;
use super::status::{SystemStatus, SystemType};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum MetricsPayload {
    Linux(LinuxMetrics),
    Docker(DockerMetrics),
    QBittorrent(QBittorrentMetrics),
    Unifi(UnifiMetrics),
    Unas(UNASMetrics),
    Empty(serde_json::Value),
}

impl Default for MetricsPayload {
    fn default() -> Self {
        MetricsPayload::Empty(serde_json::json!({}))
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemMetrics {
    pub id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub system_type: SystemType,
    #[serde(default)]
    pub status: SystemStatus,
    #[serde(default = "Utc::now")]
    pub last_updated: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    #[serde(default)]
    pub metrics: MetricsPayload,
}