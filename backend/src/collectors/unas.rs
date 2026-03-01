use async_trait::async_trait;
use chrono::Utc;

use super::ssh_connection::SshConnection;
use super::Collector;
use crate::config::UNASSystemConfig;
use crate::models::*;

pub struct UNASCollector {
    system_id: String,
    name: String,
    ssh: SshConnection,
}

impl UNASCollector {
    pub fn new(system_id: String, name: String, config: UNASSystemConfig) -> Self {
        Self {
            system_id,
            name,
            ssh: SshConnection::new(
                config.host,
                config.port,
                config.username,
                config.password,
                config.key_path,
            ),
        }
    }

    fn parse_zpool_list(output: &str) -> Vec<StoragePool> {
        let mut pools = Vec::new();
        let lines: Vec<&str> = output.trim().lines().collect();
        if lines.len() < 2 {
            return pools;
        }

        for line in &lines[1..] {
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() >= 5 {
                let name = parts[0].to_string();
                let total = parse_size(parts[1]);
                let used = parse_size(parts[2]);
                let available = parse_size(parts[3]);
                let health_candidates = ["ONLINE", "DEGRADED", "FAULTED", "OFFLINE"];
                let health = parts
                    .iter()
                    .rev()
                    .find(|p| health_candidates.contains(&p.to_uppercase().as_str()))
                    .unwrap_or(&parts[4]);

                pools.push(StoragePool {
                    name,
                    status: health.to_lowercase(),
                    total,
                    used,
                    available,
                    percent: if total > 0 {
                        (used as f64 / total as f64) * 100.0
                    } else {
                        0.0
                    },
                });
            }
        }
        pools
    }

    fn parse_disk_info(
        lsblk_output: &str,
        smartctl_outputs: &std::collections::HashMap<String, String>,
    ) -> Vec<StorageDisk> {
        let mut disks = Vec::new();

        for line in lsblk_output.lines() {
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.is_empty() {
                continue;
            }
            let name = parts[0];
            if !name.starts_with("sd") && !name.starts_with("nvme") {
                continue;
            }

            let mut disk = StorageDisk {
                name: name.to_string(),
                model: "Unknown".to_string(),
                serial: "Unknown".to_string(),
                status: "healthy".to_string(),
                temperature: None,
            };

            if let Some(smart_data) = smartctl_outputs.get(name) {
                for smart_line in smart_data.lines() {
                    if smart_line.contains("Model") || smart_line.contains("Device Model") {
                        if let Some(val) = smart_line.split(':').next_back() {
                            disk.model = val.trim().to_string();
                        }
                    } else if smart_line.contains("Serial") {
                        if let Some(val) = smart_line.split(':').next_back() {
                            disk.serial = val.trim().to_string();
                        }
                    } else if smart_line.contains("Temperature") && smart_line.contains("Celsius") {
                        let temp_parts: Vec<&str> = smart_line.split_whitespace().collect();
                        if let Some(dash_idx) = temp_parts.iter().position(|&p| p == "-") {
                            if dash_idx + 1 < temp_parts.len() {
                                let temp_str: String = temp_parts[dash_idx + 1]
                                    .chars()
                                    .take_while(|c| c.is_ascii_digit() || *c == '.')
                                    .collect();
                                if let Ok(temp) = temp_str.parse::<f64>() {
                                    disk.temperature = Some(temp);
                                }
                            }
                        } else if temp_parts.len() > 9 {
                            if let Ok(temp) = temp_parts[9].parse::<f64>() {
                                disk.temperature = Some(temp);
                            }
                        }
                    } else if smart_line.contains("SMART overall-health") {
                        disk.status = if smart_line.contains("PASSED") {
                            "healthy".to_string()
                        } else {
                            "unhealthy".to_string()
                        };
                    }
                }
            }

            disks.push(disk);
        }
        disks
    }

    fn determine_status(metrics: &UNASMetrics) -> SystemStatus {
        for pool in &metrics.pools {
            if pool.status == "faulted" {
                return SystemStatus::Critical;
            }
            if pool.status == "degraded" {
                return SystemStatus::Warning;
            }
            if pool.percent > 90.0 {
                return SystemStatus::Warning;
            }
        }
        for disk in &metrics.disks {
            if disk.status == "unhealthy" {
                return SystemStatus::Warning;
            }
            if let Some(temp) = disk.temperature {
                if temp > 50.0 {
                    return SystemStatus::Warning;
                }
            }
        }
        SystemStatus::Healthy
    }

    fn create_error_metrics(&self, error: &str) -> SystemMetrics {
        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Unas,
            status: SystemStatus::Offline,
            last_updated: Utc::now(),
            error: Some(error.to_string()),
            metrics: MetricsPayload::default(),
        }
    }
}

fn parse_size(s: &str) -> i64 {
    let s_upper = s.to_uppercase();
    let multipliers = [
        ("T", 1024_i64 * 1024 * 1024 * 1024),
        ("G", 1024_i64 * 1024 * 1024),
        ("M", 1024_i64 * 1024),
        ("K", 1024_i64),
    ];
    for (suffix, mult) in &multipliers {
        if s_upper.ends_with(suffix) {
            let num_str = &s_upper[..s_upper.len() - suffix.len()];
            if let Ok(val) = num_str.parse::<f64>() {
                return (val * *mult as f64) as i64;
            }
        }
    }
    s.parse::<f64>().map(|v| v as i64).unwrap_or(0)
}

#[async_trait]
impl Collector for UNASCollector {
    async fn collect(&mut self) -> SystemMetrics {
        let zpool_output = match self.ssh.run_command("zpool list 2>/dev/null || true").await {
            Ok(o) => o,
            Err(e) => {
                self.ssh.close().await;
                return self.create_error_metrics(&e.to_string());
            }
        };

        let df_output = match self.ssh.run_command("df -B1 2>/dev/null").await {
            Ok(o) => o,
            Err(e) => return self.create_error_metrics(&e.to_string()),
        };

        let lsblk_output = match self
            .ssh
            .run_command("lsblk -d -o NAME,SIZE,MODEL 2>/dev/null || true")
            .await
        {
            Ok(o) => o,
            Err(e) => return self.create_error_metrics(&e.to_string()),
        };

        // Get disk names
        let disk_names: Vec<String> = lsblk_output
            .lines()
            .skip(1)
            .filter_map(|line| {
                let name = line.split_whitespace().next()?;
                if name.starts_with("sd") || name.starts_with("nvme") {
                    Some(name.to_string())
                } else {
                    None
                }
            })
            .take(10)
            .collect();

        // Collect smartctl data
        let mut smartctl_outputs = std::collections::HashMap::new();
        for disk_name in &disk_names {
            let cmd = format!(
                "sudo smartctl -a /dev/{0} 2>/dev/null || smartctl -a /dev/{0} 2>/dev/null || true",
                disk_name
            );
            if let Ok(output) = self.ssh.run_command(&cmd).await {
                if !output.is_empty() {
                    smartctl_outputs.insert(disk_name.clone(), output);
                }
            }
        }

        let mut pools = Self::parse_zpool_list(&zpool_output);

        if pools.is_empty() && !df_output.is_empty() {
            let mut seen_sizes = std::collections::HashSet::new();
            let mut candidate_pools: Vec<(i64, StoragePool)> = Vec::new();

            for line in df_output.lines().skip(1) {
                let parts: Vec<&str> = line.split_whitespace().collect();
                if parts.len() >= 6 {
                    let mount = parts[5];
                    let total: i64 = parts[1].parse().unwrap_or(0);

                    let is_storage_mount = mount == "/"
                        || mount == "/volume1"
                        || mount == "/mnt"
                        || mount == "/data"
                        || (mount.starts_with("/volume/")
                            && !mount.contains("/.srv")
                            && !mount.contains("/."));

                    if is_storage_mount {
                        if total > 1_000_000_000 && seen_sizes.contains(&total) {
                            continue;
                        }
                        seen_sizes.insert(total);

                        let used: i64 = parts[2].parse().unwrap_or(0);
                        let available: i64 = parts[3].parse().unwrap_or(0);
                        let percent: f64 = parts[4]
                            .trim_end_matches('%')
                            .parse()
                            .unwrap_or(0.0);

                        candidate_pools.push((
                            total,
                            StoragePool {
                                name: mount.to_string(),
                                status: "online".to_string(),
                                total,
                                used,
                                available,
                                percent,
                            },
                        ));
                    }
                }
            }

            candidate_pools.sort_by(|a, b| b.0.cmp(&a.0));
            if let Some((_, pool)) = candidate_pools.into_iter().next() {
                pools.push(pool);
            }
        }

        let disks = Self::parse_disk_info(&lsblk_output, &smartctl_outputs);

        let total_capacity: i64 = pools.iter().map(|p| p.total).sum();
        let used_capacity: i64 = pools.iter().map(|p| p.used).sum();

        let unas_metrics = UNASMetrics {
            pools,
            disks,
            total_capacity,
            used_capacity,
        };

        let status = Self::determine_status(&unas_metrics);

        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Unas,
            status,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::Unas(unas_metrics),
        }
    }

    async fn close(&mut self) {
        self.ssh.close().await;
    }
}
