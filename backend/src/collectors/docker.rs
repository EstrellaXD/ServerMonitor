use async_trait::async_trait;
use bollard::container::{ListContainersOptions, StatsOptions};
use bollard::Docker;
use chrono::Utc;
use futures_util::TryStreamExt;

use super::Collector;
use crate::config::DockerSystemConfig;
use crate::models::*;

pub struct DockerCollector {
    system_id: String,
    name: String,
    config: DockerSystemConfig,
    client: Option<Docker>,
}

impl DockerCollector {
    pub fn new(system_id: String, name: String, config: DockerSystemConfig) -> Self {
        Self {
            system_id,
            name,
            config,
            client: None,
        }
    }

    fn get_client(&mut self) -> Result<&Docker, String> {
        if self.client.is_none() {
            let client = if let Some(ref host) = self.config.host {
                Docker::connect_with_http(host, 10, bollard::API_DEFAULT_VERSION)
                    .map_err(|e| format!("Docker connect failed: {e}"))?
            } else {
                Docker::connect_with_socket(
                    &self.config.socket,
                    10,
                    bollard::API_DEFAULT_VERSION,
                )
                .map_err(|e| format!("Docker socket connect failed: {e}"))?
            };
            self.client = Some(client);
        }
        Ok(self.client.as_ref().unwrap())
    }

    fn calculate_cpu_percent(stats: &bollard::container::Stats) -> f64 {
        let cpu_delta = stats.cpu_stats.cpu_usage.total_usage as f64
            - stats.precpu_stats.cpu_usage.total_usage as f64;
        let system_delta = stats.cpu_stats.system_cpu_usage.unwrap_or(0) as f64
            - stats.precpu_stats.system_cpu_usage.unwrap_or(0) as f64;

        if system_delta > 0.0 && cpu_delta > 0.0 {
            let cpu_count = stats.cpu_stats.online_cpus.unwrap_or(1) as f64;
            (cpu_delta / system_delta) * cpu_count * 100.0
        } else {
            0.0
        }
    }

    fn calculate_memory(stats: &bollard::container::Stats) -> (i64, i64, f64) {
        let usage = stats.memory_stats.usage.unwrap_or(0) as i64;
        let limit = stats.memory_stats.limit.unwrap_or(0) as i64;
        let percent = if limit > 0 {
            (usage as f64 / limit as f64) * 100.0
        } else {
            0.0
        };
        (usage, limit, percent)
    }

    fn determine_status(metrics: &DockerMetrics) -> SystemStatus {
        if metrics.total_count == 0 {
            return SystemStatus::Healthy;
        }

        let mut unhealthy_count = 0;
        let mut high_resource_count = 0;

        for container in &metrics.containers {
            if container.state != "running" && container.state != "paused" {
                continue;
            }
            if container.status == "unhealthy" {
                unhealthy_count += 1;
            }
            if container.cpu_percent > 90.0 || container.memory_percent > 90.0 {
                high_resource_count += 1;
            }
        }

        if unhealthy_count > 0 || metrics.stopped_count > metrics.running_count {
            return SystemStatus::Warning;
        }
        if high_resource_count > 0 {
            return SystemStatus::Warning;
        }

        SystemStatus::Healthy
    }

    fn create_error_metrics(&self, error: &str) -> SystemMetrics {
        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Docker,
            status: SystemStatus::Offline,
            last_updated: Utc::now(),
            error: Some(error.to_string()),
            metrics: MetricsPayload::default(),
        }
    }
}

#[async_trait]
impl Collector for DockerCollector {
    async fn collect(&mut self) -> SystemMetrics {
        let client = match self.get_client() {
            Ok(c) => c.clone(),
            Err(e) => return self.create_error_metrics(&e),
        };

        let containers = match client
            .list_containers(Some(ListContainersOptions::<String> {
                all: true,
                ..Default::default()
            }))
            .await
        {
            Ok(c) => c,
            Err(e) => return self.create_error_metrics(&e.to_string()),
        };

        let mut container_metrics = Vec::new();
        let mut running_count = 0i32;
        let mut stopped_count = 0i32;

        for container in &containers {
            let state = container.state.as_deref().unwrap_or("unknown").to_string();
            let name = container
                .names
                .as_ref()
                .and_then(|n| n.first())
                .map(|n| n.trim_start_matches('/').to_string())
                .unwrap_or_else(|| "unknown".to_string());
            let id = container
                .id
                .as_ref()
                .map(|id| id.chars().take(12).collect())
                .unwrap_or_else(|| "unknown".to_string());
            let image = container
                .image
                .as_deref()
                .unwrap_or("unknown")
                .to_string();

            let (cpu_percent, mem_usage, mem_limit, mem_percent) = if state == "running" {
                running_count += 1;
                if let Some(ref cid) = container.id {
                    match client
                        .stats(
                            cid,
                            Some(StatsOptions {
                                stream: false,
                                one_shot: true,
                            }),
                        )
                        .try_next()
                        .await
                    {
                        Ok(Some(stats)) => {
                            let cpu = Self::calculate_cpu_percent(&stats);
                            let (usage, limit, pct) = Self::calculate_memory(&stats);
                            (cpu, usage, limit, pct)
                        }
                        _ => (0.0, 0, 0, 0.0),
                    }
                } else {
                    (0.0, 0, 0, 0.0)
                }
            } else {
                stopped_count += 1;
                (0.0, 0, 0, 0.0)
            };

            let health_status = container
                .status
                .as_deref()
                .map(|s| {
                    if s.contains("healthy") && !s.contains("unhealthy") {
                        "healthy"
                    } else if s.contains("unhealthy") {
                        "unhealthy"
                    } else {
                        "none"
                    }
                })
                .unwrap_or("none")
                .to_string();

            container_metrics.push(ContainerMetrics {
                id,
                name,
                image,
                status: health_status,
                state,
                cpu_percent: (cpu_percent * 100.0).round() / 100.0,
                memory_usage: mem_usage,
                memory_limit: mem_limit,
                memory_percent: (mem_percent * 100.0).round() / 100.0,
            });
        }

        let docker_metrics = DockerMetrics {
            containers: container_metrics,
            running_count,
            stopped_count,
            total_count: containers.len() as i32,
        };

        let status = Self::determine_status(&docker_metrics);

        SystemMetrics {
            id: self.system_id.clone(),
            name: self.name.clone(),
            system_type: SystemType::Docker,
            status,
            last_updated: Utc::now(),
            error: None,
            metrics: MetricsPayload::Docker(docker_metrics),
        }
    }

    async fn close(&mut self) {
        self.client = None;
    }
}
