pub mod docker;
pub mod linux;
pub mod mock;
pub mod qbittorrent;
pub mod ssh_connection;
pub mod unifi;
pub mod unas;

use async_trait::async_trait;

use crate::config::{SystemConfig, SystemConfigType};
use crate::models::SystemMetrics;

#[async_trait]
pub trait Collector: Send + Sync {
    async fn collect(&mut self) -> SystemMetrics;
    async fn close(&mut self);
}

pub fn create_collector(system: &SystemConfig, mock_mode: bool) -> Option<Box<dyn Collector>> {
    if mock_mode {
        return match system.system_type.as_str() {
            "linux" => Some(Box::new(mock::MockLinuxCollector::new(
                system.id.clone(),
                system.name.clone(),
            ))),
            "docker" => Some(Box::new(mock::MockDockerCollector::new(
                system.id.clone(),
                system.name.clone(),
            ))),
            "qbittorrent" => Some(Box::new(mock::MockQBittorrentCollector::new(
                system.id.clone(),
                system.name.clone(),
            ))),
            _ => None,
        };
    }

    match &system.config {
        SystemConfigType::Linux(cfg) => Some(Box::new(linux::LinuxCollector::new(
            system.id.clone(),
            system.name.clone(),
            cfg.clone(),
        ))),
        SystemConfigType::Docker(cfg) => Some(Box::new(docker::DockerCollector::new(
            system.id.clone(),
            system.name.clone(),
            cfg.clone(),
        ))),
        SystemConfigType::QBittorrent(cfg) => Some(Box::new(qbittorrent::QBittorrentCollector::new(
            system.id.clone(),
            system.name.clone(),
            cfg.clone(),
        ))),
        SystemConfigType::Unifi(cfg) => Some(Box::new(unifi::UnifiCollector::new(
            system.id.clone(),
            system.name.clone(),
            cfg.clone(),
        ))),
        SystemConfigType::Unas(cfg) => Some(Box::new(unas::UNASCollector::new(
            system.id.clone(),
            system.name.clone(),
            cfg.clone(),
        ))),
        SystemConfigType::Unknown => None,
    }
}
