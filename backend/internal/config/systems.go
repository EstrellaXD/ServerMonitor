package config

type LinuxSystemConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password,omitempty"`
	KeyPath  string `yaml:"key_path,omitempty"`
}

type DockerSystemConfig struct {
	Host   string `yaml:"host,omitempty"`
	Socket string `yaml:"socket"`
	TLS    bool   `yaml:"tls,omitempty"`
}

type QBittorrentSystemConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type UnifiSystemConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	APIKey    string `yaml:"api_key"`
	Site      string `yaml:"site"`
	VerifySSL bool   `yaml:"verify_ssl,omitempty"`
}

type UNASSystemConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password,omitempty"`
	KeyPath  string `yaml:"key_path,omitempty"`
	APIURL   string `yaml:"api_url,omitempty"`
}
