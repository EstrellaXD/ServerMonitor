use async_trait::async_trait;
use chrono::Utc;
use reqwest::Client;

use super::Collector;
use crate::config::QBittorrentSystemConfig;
use crate::models::*;

pub struct QBittorrentCollector {
    system_id: String,
    name: String,
    config: QBittorrentSystemConfig,
}

impl QBittorrentCollector {
    pub fn new(system_id: String, name: String, config: QBittorrentSystemConfig) -> Self {
        Self {
            system_id,
            name,
            config,
        }
    }

    fn create_error_metrics(&self, error: &str) -> SystemMetrics {
        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Qbittorrent,
            status: SystemStatus::Offline,
            last_updated: Utc::now(),
            error: Some(error.to_string()),
            metrics: MetricsPayload::default(),
        }
    }

    async fn do_collect(&self) -> Result<SystemMetrics, String> {
        let client = Client::builder()
            .danger_accept_invalid_certs(true)
            .cookie_store(true)
            .timeout(std::time::Duration::from_secs(10))
            .build()
            .map_err(|e| e.to_string())?;

        // Login
        let login_resp = client
            .post(format!("{}/api/v2/auth/login", self.config.url))
            .form(&[
                ("username", self.config.username.as_str()),
                ("password", self.config.password.as_str()),
            ])
            .send()
            .await
            .map_err(|e| e.to_string())?;

        let login_text = login_resp.text().await.map_err(|e| e.to_string())?;
        if login_text != "Ok." {
            return Err("Failed to authenticate".to_string());
        }

        // Get transfer info
        let transfer_resp = client
            .get(format!("{}/api/v2/transfer/info", self.config.url))
            .send()
            .await
            .map_err(|e| e.to_string())?;
        let transfer_data: serde_json::Value =
            transfer_resp.json().await.map_err(|e| e.to_string())?;

        // Get torrents
        let torrents_resp = client
            .get(format!("{}/api/v2/torrents/info", self.config.url))
            .send()
            .await
            .map_err(|e| e.to_string())?;
        let torrents_data: Vec<serde_json::Value> =
            torrents_resp.json().await.map_err(|e| e.to_string())?;

        let download_speed = transfer_data["dl_info_speed"].as_i64().unwrap_or(0);
        let upload_speed = transfer_data["up_info_speed"].as_i64().unwrap_or(0);

        let mut active_downloads = 0i32;
        let mut active_uploads = 0i32;
        let mut torrents = Vec::new();

        for torrent in &torrents_data {
            let state = torrent["state"].as_str().unwrap_or("").to_string();

            match state.as_str() {
                "downloading" | "forcedDL" | "metaDL" | "queuedDL" => active_downloads += 1,
                "uploading" | "forcedUP" | "stalledUP" | "queuedUP" => active_uploads += 1,
                _ => {}
            }

            let eta_val = torrent["eta"].as_i64().unwrap_or(8_640_000);
            let eta = if eta_val < 8_640_000 {
                Some(eta_val)
            } else {
                None
            };

            torrents.push(TorrentInfo {
                hash: torrent["hash"].as_str().unwrap_or("").to_string(),
                name: torrent["name"].as_str().unwrap_or("Unknown").to_string(),
                size: torrent["size"].as_i64().unwrap_or(0),
                progress: torrent["progress"].as_f64().unwrap_or(0.0) * 100.0,
                download_speed: torrent["dlspeed"].as_i64().unwrap_or(0),
                upload_speed: torrent["upspeed"].as_i64().unwrap_or(0),
                state,
                eta,
            });
        }

        torrents.truncate(50);

        let qbit_metrics = QBittorrentMetrics {
            download_speed,
            upload_speed,
            active_downloads,
            active_uploads,
            total_torrents: torrents_data.len() as i32,
            torrents,
        };

        Ok(SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Qbittorrent,
            status: SystemStatus::Healthy,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::QBittorrent(qbit_metrics),
        })
    }
}

#[async_trait]
impl Collector for QBittorrentCollector {
    async fn collect(&mut self) -> SystemMetrics {
        match self.do_collect().await {
            Ok(m) => m,
            Err(e) => self.create_error_metrics(&e),
        }
    }

    async fn close(&mut self) {}
}
