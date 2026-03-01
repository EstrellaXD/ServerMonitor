use std::path::Path;

use serde::Deserialize;
use serde_yaml::Value;

use super::system_configs::*;

#[derive(Debug, Clone)]
pub struct SystemConfig {
    pub id: String,
    pub name: String,
    pub system_type: String,
    pub enabled: bool,
    pub config: SystemConfigType,
}

#[derive(Debug, Clone)]
pub enum SystemConfigType {
    Linux(LinuxSystemConfig),
    Docker(DockerSystemConfig),
    QBittorrent(QBittorrentSystemConfig),
    Unifi(UnifiSystemConfig),
    Unas(UNASSystemConfig),
    Unknown,
}

#[derive(Debug, Clone)]
pub struct AppConfig {
    pub poll_interval: u64,
    pub systems: Vec<SystemConfig>,
}

#[derive(Deserialize)]
struct RawAppConfig {
    #[serde(default = "default_poll_interval")]
    poll_interval: u64,
    #[serde(default)]
    systems: Vec<RawSystemConfig>,
}

#[derive(Deserialize)]
struct RawSystemConfig {
    id: String,
    name: String,
    #[serde(rename = "type")]
    system_type: String,
    #[serde(default = "default_enabled")]
    enabled: bool,
    #[serde(default)]
    config: Value,
}

fn default_poll_interval() -> u64 {
    5
}

fn default_enabled() -> bool {
    true
}

pub fn load_config(path: &Path) -> AppConfig {
    if !path.exists() {
        tracing::warn!("Config file not found at {:?}, using defaults", path);
        return AppConfig {
            poll_interval: 5,
            systems: vec![],
        };
    }

    let contents = std::fs::read_to_string(path).unwrap_or_default();
    let raw: RawAppConfig = serde_yaml::from_str(&contents).unwrap_or(RawAppConfig {
        poll_interval: 5,
        systems: vec![],
    });

    let systems = raw
        .systems
        .into_iter()
        .map(|raw_sys| {
            let config_type = match raw_sys.system_type.as_str() {
                "linux" => {
                    serde_yaml::from_value::<LinuxSystemConfig>(raw_sys.config.clone())
                        .map(SystemConfigType::Linux)
                        .unwrap_or(SystemConfigType::Unknown)
                }
                "docker" => {
                    serde_yaml::from_value::<DockerSystemConfig>(raw_sys.config.clone())
                        .map(SystemConfigType::Docker)
                        .unwrap_or(SystemConfigType::Unknown)
                }
                "qbittorrent" => {
                    serde_yaml::from_value::<QBittorrentSystemConfig>(raw_sys.config.clone())
                        .map(SystemConfigType::QBittorrent)
                        .unwrap_or(SystemConfigType::Unknown)
                }
                "unifi" => {
                    serde_yaml::from_value::<UnifiSystemConfig>(raw_sys.config.clone())
                        .map(SystemConfigType::Unifi)
                        .unwrap_or(SystemConfigType::Unknown)
                }
                "unas" => {
                    serde_yaml::from_value::<UNASSystemConfig>(raw_sys.config.clone())
                        .map(SystemConfigType::Unas)
                        .unwrap_or(SystemConfigType::Unknown)
                }
                _ => SystemConfigType::Unknown,
            };

            SystemConfig {
                id: raw_sys.id,
                name: raw_sys.name,
                system_type: raw_sys.system_type,
                enabled: raw_sys.enabled,
                config: config_type,
            }
        })
        .collect();

    AppConfig {
        poll_interval: raw.poll_interval,
        systems,
    }
}
