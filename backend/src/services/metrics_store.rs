use std::collections::HashMap;
use std::sync::Arc;

use tokio::sync::RwLock;

use crate::models::{MetricsUpdate, SystemMetrics};

#[derive(Clone, Default)]
pub struct MetricsStore {
    systems: Arc<RwLock<HashMap<String, SystemMetrics>>>,
}

impl MetricsStore {
    pub fn new() -> Self {
        Self::default()
    }

    pub async fn update(&self, system_id: &str, metrics: SystemMetrics) {
        self.systems
            .write()
            .await
            .insert(system_id.to_string(), metrics);
    }

    pub async fn get(&self, system_id: &str) -> Option<SystemMetrics> {
        self.systems.read().await.get(system_id).cloned()
    }

    pub async fn get_all(&self) -> HashMap<String, SystemMetrics> {
        self.systems.read().await.clone()
    }

    pub async fn clear(&self) {
        self.systems.write().await.clear();
    }

    pub async fn create_update_message(&self) -> MetricsUpdate {
        MetricsUpdate::new(self.systems.read().await.clone())
    }
}
