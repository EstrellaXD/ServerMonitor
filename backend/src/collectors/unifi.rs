use async_trait::async_trait;
use chrono::Utc;
use reqwest::Client;

use super::Collector;
use crate::config::UnifiSystemConfig;
use crate::models::*;

pub struct UnifiCollector {
    system_id: String,
    name: String,
    config: UnifiSystemConfig,
    site_uuid: Option<String>,
}

impl UnifiCollector {
    pub fn new(system_id: String, name: String, config: UnifiSystemConfig) -> Self {
        Self {
            system_id,
            name,
            config,
            site_uuid: None,
        }
    }

    fn base_url(&self) -> String {
        format!(
            "https://{}:{}/proxy/network/integration/v1",
            self.config.host, self.config.port
        )
    }

    fn site_url(&self, path: &str) -> String {
        format!(
            "{}/sites/{}/{}",
            self.base_url(),
            self.site_uuid.as_deref().unwrap_or(""),
            path
        )
    }

    async fn resolve_site_uuid(&mut self, client: &Client) -> Result<(), String> {
        if self.site_uuid.is_some() {
            return Ok(());
        }

        let resp = client
            .get(format!("{}/sites", self.base_url()))
            .header("X-API-Key", &self.config.api_key)
            .send()
            .await
            .map_err(|e| e.to_string())?;

        let data: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;

        if let Some(sites) = data["data"].as_array() {
            for site in sites {
                if site["internalReference"].as_str() == Some(&self.config.site) {
                    if let Some(id) = site["id"].as_str() {
                        self.site_uuid = Some(id.to_string());
                        return Ok(());
                    }
                }
            }
        }

        Err(format!("UniFi site '{}' not found", self.config.site))
    }

    fn determine_status(metrics: &UnifiMetrics) -> SystemStatus {
        if metrics.devices.is_empty() {
            return SystemStatus::Healthy;
        }
        let offline = metrics
            .devices
            .iter()
            .filter(|d| d.status != "online")
            .count();
        if offline > metrics.devices.len() / 2 {
            return SystemStatus::Critical;
        }
        if offline > 0 {
            return SystemStatus::Warning;
        }
        SystemStatus::Healthy
    }

    fn create_error_metrics(&self, error: &str) -> SystemMetrics {
        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Unifi,
            status: SystemStatus::Offline,
            last_updated: Utc::now(),
            error: Some(error.to_string()),
            metrics: MetricsPayload::default(),
        }
    }

    async fn do_collect(&mut self) -> Result<SystemMetrics, String> {
        let client = Client::builder()
            .danger_accept_invalid_certs(!self.config.verify_ssl)
            .timeout(std::time::Duration::from_secs(10))
            .build()
            .map_err(|e| e.to_string())?;

        self.resolve_site_uuid(&client).await?;

        // Get devices
        let devices_resp = client
            .get(self.site_url("devices"))
            .header("X-API-Key", &self.config.api_key)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        let devices_data: serde_json::Value =
            devices_resp.json().await.map_err(|e| e.to_string())?;

        // Get clients
        let clients_resp = client
            .get(self.site_url("clients"))
            .header("X-API-Key", &self.config.api_key)
            .send()
            .await
            .map_err(|e| e.to_string())?;
        let clients_data: serde_json::Value =
            clients_resp.json().await.map_err(|e| e.to_string())?;

        let mut devices = Vec::new();
        if let Some(device_list) = devices_data["data"].as_array() {
            for device in device_list {
                let state = device["state"].as_str().unwrap_or("");
                let status = if state.eq_ignore_ascii_case("ONLINE") {
                    "online"
                } else {
                    "offline"
                };

                devices.push(UnifiDevice {
                    name: device["name"]
                        .as_str()
                        .or_else(|| device["mac"].as_str())
                        .unwrap_or("Unknown")
                        .to_string(),
                    mac: device["mac"].as_str().unwrap_or("").to_string(),
                    model: device["model"].as_str().unwrap_or("Unknown").to_string(),
                    status: status.to_string(),
                    uptime: device["uptime"].as_i64().unwrap_or(0),
                });
            }
        }

        let mut guests_count = 0i32;
        let mut total_clients = 0i32;
        if let Some(client_list) = clients_data["data"].as_array() {
            total_clients = client_list.len() as i32;
            for c in client_list {
                if c["isGuest"].as_bool().unwrap_or(false) {
                    guests_count += 1;
                }
            }
        }

        let unifi_metrics = UnifiMetrics {
            devices,
            clients_count: total_clients - guests_count,
            guests_count,
            wan_download: 0.0,
            wan_upload: 0.0,
        };

        let status = Self::determine_status(&unifi_metrics);

        Ok(SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Unifi,
            status,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::Unifi(unifi_metrics),
        })
    }
}

#[async_trait]
impl Collector for UnifiCollector {
    async fn collect(&mut self) -> SystemMetrics {
        match self.do_collect().await {
            Ok(m) => m,
            Err(e) => self.create_error_metrics(&e),
        }
    }

    async fn close(&mut self) {}
}
