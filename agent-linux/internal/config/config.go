package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "/etc/backupsmc/agent.yaml"

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Sources     SourcesConfig     `yaml:"sources"`
	Destination DestinationConfig `yaml:"destination"`
	Schedule    ScheduleConfig    `yaml:"schedule"`
	Retention   RetentionConfig   `yaml:"retention"`
	Log         LogConfig         `yaml:"log"`
}

type ServerConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type SourcesConfig struct {
	Files       FilesConfig       `yaml:"files"`
	Databases   DatabasesConfig   `yaml:"databases"`
	VMs         VMsConfig         `yaml:"vms"`
}

type FilesConfig struct {
	Enabled bool     `yaml:"enabled"`
	Paths   []string `yaml:"paths"`
	Exclude []string `yaml:"exclude"`
}

type DatabasesConfig struct {
	PostgreSQL    *PostgreSQLConfig    `yaml:"postgresql,omitempty"`
	MySQL         *MySQLConfig         `yaml:"mysql,omitempty"`
	MongoDB       *MongoDBConfig       `yaml:"mongodb,omitempty"`
	Redis         *RedisConfig         `yaml:"redis,omitempty"`
	SQLite        *SQLiteConfig        `yaml:"sqlite,omitempty"`
	Elasticsearch *ElasticsearchConfig `yaml:"elasticsearch,omitempty"`
}

type PostgreSQLConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Host       string   `yaml:"host"`
	Port       int      `yaml:"port"`
	User       string   `yaml:"user"`
	Password   string   `yaml:"password"`
	Databases  []string `yaml:"databases"` // vacío = todas
}

type MySQLConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Host      string   `yaml:"host"`
	Port      int      `yaml:"port"`
	User      string   `yaml:"user"`
	Password  string   `yaml:"password"`
	Databases []string `yaml:"databases"`
}

type MongoDBConfig struct {
	Enabled    bool   `yaml:"enabled"`
	URI        string `yaml:"uri"`
	Databases  []string `yaml:"databases"`
}

type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DataDir  string `yaml:"data_dir"`
}

type SQLiteConfig struct {
	Enabled bool     `yaml:"enabled"`
	Files   []string `yaml:"files"`
}

type ElasticsearchConfig struct {
	Enabled    bool     `yaml:"enabled"`
	URL        string   `yaml:"url"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
	Indices    []string `yaml:"indices"`
}

type VMsConfig struct {
	Proxmox *ProxmoxConfig `yaml:"proxmox,omitempty"`
	VMware  *VMwareConfig  `yaml:"vmware,omitempty"`
	KVM     *KVMConfig     `yaml:"kvm,omitempty"`
}

type ProxmoxConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Node     string `yaml:"node"`
	VMIDs    []int  `yaml:"vm_ids"` // vacío = todas
}

type VMwareConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	VMNames  []string `yaml:"vm_names"`
}

type KVMConfig struct {
	Enabled bool     `yaml:"enabled"`
	VMNames []string `yaml:"vm_names"`
}

type DestinationConfig struct {
	Type  string      `yaml:"type"` // local|sftp|s3|nfs|smb
	Local *LocalDest  `yaml:"local,omitempty"`
	SFTP  *SFTPDest   `yaml:"sftp,omitempty"`
	S3    *S3Dest     `yaml:"s3,omitempty"`
	NFS   *NFSDest    `yaml:"nfs,omitempty"`
	SMB   *SMBDest    `yaml:"smb,omitempty"`
}

type LocalDest struct {
	Path string `yaml:"path"`
}

type SFTPDest struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"key_file"`
	Path     string `yaml:"path"`
}

type S3Dest struct {
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Prefix    string `yaml:"prefix"`
}

type NFSDest struct {
	Server     string `yaml:"server"`
	Export     string `yaml:"export"`
	MountPoint string `yaml:"mount_point"`
	Options    string `yaml:"options"`
}

type SMBDest struct {
	Share      string `yaml:"share"`
	MountPoint string `yaml:"mount_point"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Domain     string `yaml:"domain"`
}

type ScheduleConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Cron         string `yaml:"cron"`          // "0 2 * * *"
	FullOnWeekday int   `yaml:"full_on_weekday"` // 0=Dom, -1=nunca
}

type RetentionConfig struct {
	Days    int    `yaml:"days"`
	GFS     *GFSConfig `yaml:"gfs,omitempty"`
}

type GFSConfig struct {
	Daily   int `yaml:"daily"`
	Weekly  int `yaml:"weekly"`
	Monthly int `yaml:"monthly"`
}

type LogConfig struct {
	Level string `yaml:"level"` // debug|info|warn|error
	File  string `yaml:"file"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leer config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsear config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0640)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
