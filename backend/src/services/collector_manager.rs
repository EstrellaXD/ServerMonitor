use std::collections::HashMap;

use tokio::sync::broadcast;
use tokio::task::JoinHandle;

use crate::collectors::{create_collector, Collector};
use crate::config::AppConfig;
use crate::services::metrics_store::MetricsStore;

pub struct CollectorManager {
    collectors: HashMap<String, Box<dyn Collector>>,
    poll_interval: u64,
    task: Option<JoinHandle<()>>,
    store: MetricsStore,
    tx: broadcast::Sender<String>,
    mock_mode: bool,
}

impl CollectorManager {
    pub fn new(store: MetricsStore, tx: broadcast::Sender<String>, mock_mode: bool) -> Self {
        Self {
            collectors: HashMap::new(),
            poll_interval: 5,
            task: None,
            store,
            tx,
            mock_mode,
        }
    }

    pub fn load_config(&mut self, config: &AppConfig) {
        self.poll_interval = config.poll_interval;
        self.collectors.clear();

        for system in &config.systems {
            if !system.enabled {
                continue;
            }
            if let Some(collector) = create_collector(system, self.mock_mode) {
                self.collectors.insert(system.id.clone(), collector);
            }
        }
    }

    pub fn collector_count(&self) -> usize {
        self.collectors.len()
    }

    pub fn collector_ids(&self) -> Vec<String> {
        self.collectors.keys().cloned().collect()
    }

    pub async fn start(&mut self) {
        if self.task.is_some() {
            return;
        }

        // Take ownership of collectors for the spawned task
        let mut collectors: HashMap<String, Box<dyn Collector>> =
            std::mem::take(&mut self.collectors);
        let store = self.store.clone();
        let tx = self.tx.clone();
        let poll_interval = self.poll_interval;

        let handle = tokio::spawn(async move {
            tracing::info!(
                "Collection loop started, poll_interval={}, collectors={}",
                poll_interval,
                collectors.len()
            );

            loop {
                tracing::debug!("Running collection for {} collectors...", collectors.len());

                // Collect from all collectors
                for (id, collector) in collectors.iter_mut() {
                    let metrics = collector.collect().await;
                    tracing::debug!("[{}] Collection OK: status={:?}", id, metrics.status);
                    store.update(id, metrics).await;
                }

                // Broadcast update
                let update = store.create_update_message().await;
                if let Ok(json) = serde_json::to_string(&update) {
                    // Ignore send errors (no receivers)
                    let _ = tx.send(json);
                }

                tokio::time::sleep(std::time::Duration::from_secs(poll_interval)).await;
            }
        });

        self.task = Some(handle);
    }

    pub async fn stop(&mut self) {
        if let Some(handle) = self.task.take() {
            handle.abort();
            let _ = handle.await;
        }
        // Close any remaining collectors
        for (_, collector) in self.collectors.iter_mut() {
            collector.close().await;
        }
    }
}
