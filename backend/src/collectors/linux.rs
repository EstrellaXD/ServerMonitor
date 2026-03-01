use std::collections::HashMap;

use async_trait::async_trait;
use chrono::Utc;

use super::ssh_connection::SshConnection;
use super::Collector;
use crate::config::LinuxSystemConfig;
use crate::models::*;

pub struct LinuxCollector {
    system_id: String,
    name: String,
    ssh: SshConnection,
    prev_cpu_stats: Option<HashMap<String, CpuStat>>,
    prev_net_stats: Option<NetStat>,
    prev_time: Option<f64>,
}

#[derive(Clone)]
struct CpuStat {
    user: i64,
    nice: i64,
    system: i64,
    idle: i64,
    iowait: i64,
    irq: i64,
    softirq: i64,
}

impl CpuStat {
    fn total(&self) -> i64 {
        self.user + self.nice + self.system + self.idle + self.iowait + self.irq + self.softirq
    }
    fn idle_total(&self) -> i64 {
        self.idle + self.iowait
    }
}

#[derive(Clone)]
struct NetStat {
    bytes_recv: i64,
    bytes_sent: i64,
}

impl LinuxCollector {
    pub fn new(system_id: String, name: String, config: LinuxSystemConfig) -> Self {
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
            prev_cpu_stats: None,
            prev_net_stats: None,
            prev_time: None,
        }
    }

    fn parse_combined_output(output: &str) -> HashMap<String, String> {
        let mut sections = HashMap::new();
        let mut current_section: Option<String> = None;
        let mut current_lines = Vec::new();

        for line in output.lines() {
            if line.starts_with("===") && line.ends_with("===") {
                if let Some(ref section) = current_section {
                    sections.insert(section.clone(), current_lines.join("\n"));
                }
                current_section = Some(line.trim_matches('=').to_string());
                current_lines.clear();
            } else if current_section.is_some() {
                current_lines.push(line.to_string());
            }
        }
        if let Some(section) = current_section {
            sections.insert(section, current_lines.join("\n"));
        }
        sections
    }

    fn parse_cpu_stats(stat_output: &str) -> HashMap<String, CpuStat> {
        let mut stats = HashMap::new();
        for line in stat_output.lines() {
            if line.starts_with("cpu") {
                let parts: Vec<&str> = line.split_whitespace().collect();
                if parts.len() >= 8 {
                    let name = parts[0].to_string();
                    let values: Vec<i64> = parts[1..8]
                        .iter()
                        .filter_map(|v| v.parse().ok())
                        .collect();
                    if values.len() >= 7 {
                        stats.insert(
                            name,
                            CpuStat {
                                user: values[0],
                                nice: values[1],
                                system: values[2],
                                idle: values[3],
                                iowait: values[4],
                                irq: values[5],
                                softirq: values[6],
                            },
                        );
                    }
                }
            }
        }
        stats
    }

    fn calculate_cpu_percent(
        current: &HashMap<String, CpuStat>,
        previous: &Option<HashMap<String, CpuStat>>,
    ) -> (f64, Vec<f64>) {
        let prev = match previous {
            Some(p) => p,
            None => return (0.0, vec![]),
        };

        let calc = |cur: &CpuStat, prv: &CpuStat| -> f64 {
            let total_diff = cur.total() - prv.total();
            let idle_diff = cur.idle_total() - prv.idle_total();
            if total_diff == 0 {
                return 0.0;
            }
            ((total_diff - idle_diff) as f64 / total_diff as f64) * 100.0
        };

        let total_percent = match (current.get("cpu"), prev.get("cpu")) {
            (Some(c), Some(p)) => calc(c, p),
            _ => 0.0,
        };

        let mut core_percents = Vec::new();
        let mut keys: Vec<&String> = current
            .keys()
            .filter(|k| k.starts_with("cpu") && *k != "cpu")
            .collect();
        keys.sort();
        for key in keys {
            if let (Some(c), Some(p)) = (current.get(key), prev.get(key)) {
                core_percents.push(calc(c, p));
            }
        }

        (total_percent, core_percents)
    }

    fn parse_meminfo(meminfo: &str) -> MemoryMetrics {
        let mut info = HashMap::new();
        for line in meminfo.lines() {
            if let Some((key, value_part)) = line.split_once(':') {
                let value_str = value_part.split_whitespace().next().unwrap_or("0");
                if let Ok(val) = value_str.parse::<i64>() {
                    info.insert(key.to_string(), val * 1024);
                }
            }
        }

        let total = *info.get("MemTotal").unwrap_or(&0);
        let available = *info.get("MemAvailable").unwrap_or(&0);
        let used = total - available;

        MemoryMetrics {
            total,
            used,
            available,
            percent: if total > 0 {
                (used as f64 / total as f64) * 100.0
            } else {
                0.0
            },
        }
    }

    fn parse_df(df_output: &str) -> Vec<DiskMetrics> {
        let mut disks = Vec::new();
        for line in df_output.lines().skip(1) {
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() >= 6 && parts[0].starts_with('/') {
                if let (Ok(total), Ok(used), Ok(free)) = (
                    parts[1].parse::<i64>(),
                    parts[2].parse::<i64>(),
                    parts[3].parse::<i64>(),
                ) {
                    let percent = parts[4]
                        .trim_end_matches('%')
                        .parse::<f64>()
                        .unwrap_or(0.0);
                    disks.push(DiskMetrics {
                        mount: parts[5].to_string(),
                        total: total * 1024,
                        used: used * 1024,
                        free: free * 1024,
                        percent,
                    });
                }
            }
        }
        disks
    }

    fn parse_net_stats(net_output: &str) -> NetStat {
        let mut stats = NetStat {
            bytes_recv: 0,
            bytes_sent: 0,
        };

        for line in net_output.lines().skip(2) {
            if let Some((iface_part, values_part)) = line.split_once(':') {
                let iface = iface_part.trim();
                if iface == "lo" || iface == "docker0" || iface.starts_with("veth") {
                    continue;
                }
                let values: Vec<&str> = values_part.split_whitespace().collect();
                if values.len() > 8 {
                    if let (Ok(recv), Ok(sent)) =
                        (values[0].parse::<i64>(), values[8].parse::<i64>())
                    {
                        stats.bytes_recv += recv;
                        stats.bytes_sent += sent;
                    }
                }
            }
        }
        stats
    }

    fn parse_sensors(sensors_output: &str) -> Vec<TemperatureMetrics> {
        let mut temps = Vec::new();
        for line in sensors_output.lines() {
            if line.contains(':') && line.contains("°C") {
                if let Some((label_part, temp_part)) = line.split_once(':') {
                    let label = label_part.trim().to_string();
                    let temp_str = temp_part.split("°C").next().unwrap_or("").trim();
                    let temp_str = temp_str.strip_prefix('+').unwrap_or(temp_str);
                    if let Ok(temp) = temp_str.parse::<f64>() {
                        temps.push(TemperatureMetrics {
                            label,
                            current: temp,
                            high: None,
                            critical: None,
                        });
                    }
                }
            }
        }
        temps
    }

    fn parse_thermal_zones(thermal_output: &str) -> Vec<TemperatureMetrics> {
        let mut temps = Vec::new();
        for line in thermal_output.lines() {
            if let Some((label, temp_str)) = line.split_once(':') {
                let label = label.trim().to_string();
                if let Ok(temp_milli) = temp_str.trim().parse::<f64>() {
                    temps.push(TemperatureMetrics {
                        label,
                        current: temp_milli / 1000.0,
                        high: None,
                        critical: None,
                    });
                }
            }
        }
        temps
    }

    fn parse_uptime(uptime_output: &str) -> i64 {
        uptime_output
            .split_whitespace()
            .next()
            .and_then(|s| s.parse::<f64>().ok())
            .map(|f| f as i64)
            .unwrap_or(0)
    }

    fn determine_status(metrics: &LinuxMetrics) -> SystemStatus {
        if metrics.cpu.percent > 90.0 || metrics.memory.percent > 95.0 {
            return SystemStatus::Critical;
        }
        if metrics.cpu.percent > 75.0 || metrics.memory.percent > 85.0 {
            return SystemStatus::Warning;
        }
        for disk in &metrics.disks {
            if disk.percent > 95.0 {
                return SystemStatus::Critical;
            }
            if disk.percent > 85.0 {
                return SystemStatus::Warning;
            }
        }
        for temp in &metrics.temperatures {
            if let Some(critical) = temp.critical {
                if temp.current >= critical {
                    return SystemStatus::Critical;
                }
            }
            if let Some(high) = temp.high {
                if temp.current >= high {
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
            system_type: SystemType::Linux,
            status: SystemStatus::Offline,
            last_updated: Utc::now(),
            error: Some(error.to_string()),
            metrics: MetricsPayload::default(),
        }
    }
}

#[async_trait]
impl Collector for LinuxCollector {
    async fn collect(&mut self) -> SystemMetrics {
        let current_time = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_secs_f64())
            .unwrap_or(0.0);

        let combined_cmd = concat!(
            r#"echo "===STAT===" && cat /proc/stat && "#,
            r#"echo "===MEMINFO===" && cat /proc/meminfo && "#,
            r#"echo "===DF===" && df -B1 && "#,
            r#"echo "===NETDEV===" && cat /proc/net/dev && "#,
            r#"echo "===UPTIME===" && cat /proc/uptime && "#,
            r#"echo "===LOADAVG===" && cat /proc/loadavg && "#,
            r#"echo "===SENSORS===" && (sensors 2>/dev/null || true) && "#,
            r#"echo "===THERMAL===" && (for zone in /sys/class/thermal/thermal_zone*/; do "#,
            r#"name=$(cat ${zone}type 2>/dev/null || echo zone${zone##*zone}); "#,
            r#"temp=$(cat ${zone}temp 2>/dev/null); "#,
            r#"[ -n "$temp" ] && echo "$name:$temp"; done 2>/dev/null || true)"#
        );

        let output = match self.ssh.run_command(combined_cmd).await {
            Ok(o) => o,
            Err(e) => {
                self.ssh.close().await;
                return self.create_error_metrics(&e.to_string());
            }
        };

        let sections = Self::parse_combined_output(&output);

        let stat = sections.get("STAT").map(|s| s.as_str()).unwrap_or("");
        let meminfo = sections.get("MEMINFO").map(|s| s.as_str()).unwrap_or("");
        let df = sections.get("DF").map(|s| s.as_str()).unwrap_or("");
        let netdev = sections.get("NETDEV").map(|s| s.as_str()).unwrap_or("");
        let uptime_str = sections.get("UPTIME").map(|s| s.as_str()).unwrap_or("");
        let loadavg_str = sections.get("LOADAVG").map(|s| s.as_str()).unwrap_or("");
        let sensors = sections.get("SENSORS").map(|s| s.as_str()).unwrap_or("");
        let thermal = sections.get("THERMAL").map(|s| s.as_str()).unwrap_or("");

        let cpu_stats = Self::parse_cpu_stats(stat);
        let (cpu_percent, core_percents) =
            Self::calculate_cpu_percent(&cpu_stats, &self.prev_cpu_stats);
        self.prev_cpu_stats = Some(cpu_stats);

        let load_avg: Vec<f64> = loadavg_str
            .split_whitespace()
            .take(3)
            .filter_map(|s| s.parse().ok())
            .collect();

        let memory = Self::parse_meminfo(meminfo);
        let disks = Self::parse_df(df);

        let net_stats = Self::parse_net_stats(netdev);
        let (upload_rate, download_rate) = match (&self.prev_net_stats, self.prev_time) {
            (Some(prev), Some(prev_time)) => {
                let time_diff = current_time - prev_time;
                if time_diff > 0.0 {
                    (
                        (net_stats.bytes_sent - prev.bytes_sent) as f64 / time_diff,
                        (net_stats.bytes_recv - prev.bytes_recv) as f64 / time_diff,
                    )
                } else {
                    (0.0, 0.0)
                }
            }
            _ => (0.0, 0.0),
        };
        self.prev_net_stats = Some(net_stats.clone());
        self.prev_time = Some(current_time);

        let network = NetworkMetrics {
            bytes_sent: net_stats.bytes_sent,
            bytes_recv: net_stats.bytes_recv,
            upload_rate,
            download_rate,
        };

        let temperatures = if !sensors.trim().is_empty() {
            Self::parse_sensors(sensors)
        } else if !thermal.trim().is_empty() {
            Self::parse_thermal_zones(thermal)
        } else {
            vec![]
        };

        let uptime_seconds = Self::parse_uptime(uptime_str);

        let linux_metrics = LinuxMetrics {
            cpu: CpuMetrics {
                percent: cpu_percent,
                cores: core_percents,
                load_avg,
            },
            memory,
            disks,
            network,
            temperatures,
            uptime: uptime_seconds,
        };

        let status = Self::determine_status(&linux_metrics);

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

    async fn close(&mut self) {
        self.ssh.close().await;
    }
}
