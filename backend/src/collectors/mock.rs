use async_trait::async_trait;
use chrono::Utc;
use rand::Rng;

use super::Collector;
use crate::models::*;

// --- Mock Linux ---

pub struct MockLinuxCollector {
    system_id: String,
    name: String,
    base_cpu: f64,
    base_mem: f64,
    uptime: i64,
}

impl MockLinuxCollector {
    pub fn new(system_id: String, name: String) -> Self {
        let mut rng = rand::thread_rng();
        Self {
            system_id,
            name,
            base_cpu: rng.gen_range(10.0..40.0),
            base_mem: rng.gen_range(30.0..60.0),
            uptime: rng.gen_range(86400..864000),
        }
    }
}

#[async_trait]
impl Collector for MockLinuxCollector {
    async fn collect(&mut self) -> SystemMetrics {
        let mut rng = rand::thread_rng();
        let cpu_percent = (self.base_cpu + rng.gen_range(-10.0..15.0)).clamp(0.0, 100.0);
        let mem_percent = (self.base_mem + rng.gen_range(-5.0..10.0)).clamp(0.0, 100.0);

        let total_mem: i64 = 16 * 1024 * 1024 * 1024;
        let used_mem = (total_mem as f64 * mem_percent / 100.0) as i64;

        let core_percents: Vec<f64> = (0..8)
            .map(|_| (cpu_percent + rng.gen_range(-20.0..20.0)).clamp(0.0, 100.0))
            .collect();

        let linux_metrics = LinuxMetrics {
            cpu: CpuMetrics {
                percent: cpu_percent,
                cores: core_percents,
                load_avg: vec![
                    cpu_percent / 25.0,
                    cpu_percent / 30.0,
                    cpu_percent / 35.0,
                ],
            },
            memory: MemoryMetrics {
                total: total_mem,
                used: used_mem,
                available: total_mem - used_mem,
                percent: mem_percent,
            },
            disks: vec![
                DiskMetrics {
                    mount: "/".to_string(),
                    total: 500 * 1024 * 1024 * 1024,
                    used: (500.0 * 1024.0 * 1024.0 * 1024.0 * 0.45) as i64,
                    free: (500.0 * 1024.0 * 1024.0 * 1024.0 * 0.55) as i64,
                    percent: 45.0 + rng.gen_range(-2.0..2.0),
                },
                DiskMetrics {
                    mount: "/home".to_string(),
                    total: 1000 * 1024 * 1024 * 1024,
                    used: (1000.0 * 1024.0 * 1024.0 * 1024.0 * 0.62) as i64,
                    free: (1000.0 * 1024.0 * 1024.0 * 1024.0 * 0.38) as i64,
                    percent: 62.0 + rng.gen_range(-2.0..2.0),
                },
            ],
            network: NetworkMetrics {
                bytes_sent: rng.gen_range(1_000_000_000..5_000_000_000),
                bytes_recv: rng.gen_range(5_000_000_000..20_000_000_000),
                upload_rate: rng.gen_range(100_000.0..5_000_000.0),
                download_rate: rng.gen_range(500_000.0..10_000_000.0),
            },
            temperatures: vec![
                TemperatureMetrics {
                    label: "CPU".to_string(),
                    current: 45.0 + rng.gen_range(-5.0..15.0),
                    high: None,
                    critical: None,
                },
                TemperatureMetrics {
                    label: "GPU".to_string(),
                    current: 40.0 + rng.gen_range(-5.0..10.0),
                    high: None,
                    critical: None,
                },
            ],
            uptime: self.uptime,
        };

        self.uptime += 5;

        let status = if linux_metrics.cpu.percent > 90.0 || linux_metrics.memory.percent > 95.0 {
            SystemStatus::Critical
        } else if linux_metrics.cpu.percent > 75.0 || linux_metrics.memory.percent > 85.0 {
            SystemStatus::Warning
        } else {
            SystemStatus::Healthy
        };

        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Linux,
            status,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::Linux(linux_metrics),
        }
    }

    async fn close(&mut self) {}
}

// --- Mock Docker ---

pub struct MockDockerCollector {
    system_id: String,
    name: String,
    containers: Vec<(&'static str, &'static str)>,
}

impl MockDockerCollector {
    pub fn new(system_id: String, name: String) -> Self {
        Self {
            system_id,
            name,
            containers: vec![
                ("nginx", "running"),
                ("postgres", "running"),
                ("redis", "running"),
                ("api-server", "running"),
                ("worker", "running"),
                ("monitoring", "stopped"),
            ],
        }
    }
}

#[async_trait]
impl Collector for MockDockerCollector {
    async fn collect(&mut self) -> SystemMetrics {
        let mut rng = rand::thread_rng();
        let mut container_metrics = Vec::new();
        let mut running = 0i32;
        let mut stopped = 0i32;

        for (name, state) in &self.containers {
            let is_running = *state == "running";
            if is_running {
                running += 1;
            } else {
                stopped += 1;
            }

            container_metrics.push(ContainerMetrics {
                id: format!("mock_{name}"),
                name: name.to_string(),
                image: format!("{name}:latest"),
                state: state.to_string(),
                status: if is_running {
                    "Up 3 days".to_string()
                } else {
                    "Exited (0) 2 hours ago".to_string()
                },
                cpu_percent: if is_running {
                    rng.gen_range(0.5..15.0)
                } else {
                    0.0
                },
                memory_percent: if is_running {
                    rng.gen_range(1.0..25.0)
                } else {
                    0.0
                },
                memory_usage: if is_running {
                    rng.gen_range(50_000_000..500_000_000)
                } else {
                    0
                },
                memory_limit: 1024 * 1024 * 1024,
            });
        }

        let status = if stopped > running {
            SystemStatus::Warning
        } else {
            SystemStatus::Healthy
        };

        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Docker,
            status,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::Docker(DockerMetrics {
                containers: container_metrics,
                total_count: self.containers.len() as i32,
                running_count: running,
                stopped_count: stopped,
            }),
        }
    }

    async fn close(&mut self) {}
}

// --- Mock qBittorrent ---

pub struct MockQBittorrentCollector {
    system_id: String,
    name: String,
    torrents: Vec<MockTorrent>,
}

struct MockTorrent {
    name: &'static str,
    size: i64,
    progress: f64,
    state: String,
}

impl MockQBittorrentCollector {
    pub fn new(system_id: String, name: String) -> Self {
        Self {
            system_id,
            name,
            torrents: vec![
                MockTorrent {
                    name: "Ubuntu 24.04 LTS",
                    size: (4.5 * 1024.0 * 1024.0 * 1024.0) as i64,
                    progress: 100.0,
                    state: "stalledUP".to_string(),
                },
                MockTorrent {
                    name: "Debian 12.0",
                    size: (3.8 * 1024.0 * 1024.0 * 1024.0) as i64,
                    progress: 100.0,
                    state: "uploading".to_string(),
                },
                MockTorrent {
                    name: "Fedora 40",
                    size: (2.1 * 1024.0 * 1024.0 * 1024.0) as i64,
                    progress: 78.5,
                    state: "downloading".to_string(),
                },
                MockTorrent {
                    name: "Arch Linux",
                    size: (850.0 * 1024.0 * 1024.0) as i64,
                    progress: 45.2,
                    state: "downloading".to_string(),
                },
                MockTorrent {
                    name: "Linux Mint 21.3",
                    size: (2.8 * 1024.0 * 1024.0 * 1024.0) as i64,
                    progress: 92.1,
                    state: "downloading".to_string(),
                },
            ],
        }
    }
}

#[async_trait]
impl Collector for MockQBittorrentCollector {
    async fn collect(&mut self) -> SystemMetrics {
        let mut rng = rand::thread_rng();
        let mut torrent_infos = Vec::new();
        let mut active_downloads = 0i32;
        let mut total_download_speed: i64 = 0;
        let mut total_upload_speed: i64 = 0;

        for (idx, t) in self.torrents.iter_mut().enumerate() {
            if t.progress < 100.0 && t.state == "downloading" {
                t.progress = (t.progress + rng.gen_range(0.1..0.5)).min(100.0);
                if t.progress >= 100.0 {
                    t.state = "stalledUP".to_string();
                    t.progress = 100.0;
                }
            }

            let is_downloading = t.state == "downloading" || t.state == "forcedDL";
            let is_uploading =
                t.state == "uploading" || t.state == "stalledUP" || t.state == "forcedUP";

            let dl_speed = if is_downloading {
                (rng.gen_range(1.0..10.0) * 1024.0 * 1024.0) as i64
            } else {
                0
            };
            let ul_speed = if is_uploading {
                (rng.gen_range(0.1..2.0) * 1024.0 * 1024.0) as i64
            } else {
                0
            };

            if is_downloading {
                active_downloads += 1;
                total_download_speed += dl_speed;
            }
            total_upload_speed += ul_speed;

            let eta = if is_downloading && dl_speed > 0 {
                let remaining = (t.size as f64 * (100.0 - t.progress) / 100.0) as i64;
                Some(remaining / dl_speed)
            } else {
                None
            };

            torrent_infos.push(TorrentInfo {
                hash: format!("{:032x}", t.name.len() as u128 * 0xdeadbeef + idx as u128),
                name: t.name.to_string(),
                size: t.size,
                progress: t.progress,
                state: t.state.clone(),
                download_speed: dl_speed,
                upload_speed: ul_speed,
                eta,
            });
        }

        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Qbittorrent,
            status: SystemStatus::Healthy,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::QBittorrent(QBittorrentMetrics {
                download_speed: total_download_speed,
                upload_speed: total_upload_speed,
                active_downloads,
                active_uploads: 0,
                total_torrents: torrent_infos.len() as i32,
                torrents: torrent_infos,
            }),
        }
    }

    async fn close(&mut self) {}
}
