use std::collections::HashMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use super::system_metrics::SystemMetrics;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsUpdate {
    #[serde(rename = "type")]
    pub msg_type: String,
    pub timestamp: DateTime<Utc>,
    pub systems: HashMap<String, SystemMetrics>,
}

impl MetricsUpdate {
    pub fn new(systems: HashMap<String, SystemMetrics>) -> Self {
        Self {
            msg_type: "metrics_update".to_string(),
            timestamp: Utc::now(),
            systems,
        }
    }
}
