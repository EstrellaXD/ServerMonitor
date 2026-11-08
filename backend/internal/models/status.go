package models

type SystemStatus string

const (
	StatusHealthy  SystemStatus = "healthy"
	StatusWarning  SystemStatus = "warning"
	StatusCritical SystemStatus = "critical"
	StatusOffline  SystemStatus = "offline"
)

type SystemType string

const (
	TypeLinux       SystemType = "linux"
	TypeDocker      SystemType = "docker"
	TypeQBittorrent SystemType = "qbittorrent"
	TypeUnifi       SystemType = "unifi"
	TypeUnas        SystemType = "unas"
)
