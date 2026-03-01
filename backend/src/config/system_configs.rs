use serde::Deserialize;

#[derive(Debug, Clone, Deserialize)]
pub struct LinuxSystemConfig {
    pub host: String,
    #[serde(default = "default_ssh_port")]
    pub port: u16,
    pub username: String,
    pub password: Option<String>,
    pub key_path: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct DockerSystemConfig {
    pub host: Option<String>,
    #[serde(default = "default_docker_socket")]
    pub socket: String,
    #[serde(default)]
    pub tls: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct QBittorrentSystemConfig {
    pub url: String,
    pub username: String,
    pub password: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct UnifiSystemConfig {
    pub host: String,
    #[serde(default = "default_unifi_port")]
    pub port: u16,
    pub api_key: String,
    #[serde(default = "default_site")]
    pub site: String,
    #[serde(default)]
    pub verify_ssl: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct UNASSystemConfig {
    pub host: String,
    #[serde(default = "default_ssh_port")]
    pub port: u16,
    pub username: String,
    pub password: Option<String>,
    pub key_path: Option<String>,
    pub api_url: Option<String>,
}

fn default_ssh_port() -> u16 {
    22
}

fn default_docker_socket() -> String {
    "unix:///var/run/docker.sock".to_string()
}

fn default_unifi_port() -> u16 {
    443
}

fn default_site() -> String {
    "default".to_string()
}
