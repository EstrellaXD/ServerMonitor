use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SystemStatus {
    Healthy,
    Warning,
    Critical,
    #[default]
    Offline,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SystemType {
    Linux,
    Docker,
    Qbittorrent,
    Unifi,
    Unas,
}

impl std::fmt::Display for SystemType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SystemType::Linux => write!(f, "linux"),
            SystemType::Docker => write!(f, "docker"),
            SystemType::Qbittorrent => write!(f, "qbittorrent"),
            SystemType::Unifi => write!(f, "unifi"),
            SystemType::Unas => write!(f, "unas"),
        }
    }
}
