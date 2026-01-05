package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/qualys/dspm/internal/models"
)

type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Redis         RedisConfig         `yaml:"redis"`
	Neo4j         Neo4jConfig         `yaml:"neo4j"`
	Scanner       ScannerConfig       `yaml:"scanner"`
	AWS           AWSConfig           `yaml:"aws"`
	Azure         AzureConfig         `yaml:"azure"`
	GCP           GCPConfig           `yaml:"gcp"`
	Auth          AuthConfig          `yaml:"auth"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type AuthConfig struct {
	JWTSecret          string        `yaml:"jwt_secret"`
	AccessTokenExpiry  time.Duration `yaml:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `yaml:"refresh_token_expiry"`
}

type NotificationsConfig struct {
	MinSeverity models.Sensitivity `yaml:"min_severity"`
	Slack       SlackNotifyConfig  `yaml:"slack"`
	Email       EmailNotifyConfig  `yaml:"email"`
}

type SlackNotifyConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

type EmailNotifyConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	CORSAllowOrigin string        `yaml:"cors_allow_origin"`
}

type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	SSLMode      string `yaml:"ssl_mode"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type ScannerConfig struct {
	Workers          int           `yaml:"workers"`
	BatchSize        int           `yaml:"batch_size"`
	ScanTimeout      time.Duration `yaml:"scan_timeout"`
	MaxFileSize      int64         `yaml:"max_file_size"`
	SampleSize       int64         `yaml:"sample_size"`
	FilesPerBucket   int           `yaml:"files_per_bucket"`
	RandomSamplePct  float64       `yaml:"random_sample_pct"`
	EnabledProviders []string      `yaml:"enabled_providers"`
}

type AWSConfig struct {
	Region          string `yaml:"region"`
	AssumeRoleARN   string `yaml:"assume_role_arn"`
	ExternalID      string `yaml:"external_id"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

type AzureConfig struct {
	TenantID       string `yaml:"tenant_id"`
	ClientID       string `yaml:"client_id"`
	ClientSecret   string `yaml:"client_secret"`
	SubscriptionID string `yaml:"subscription_id"`
}

type GCPConfig struct {
	ProjectID       string `yaml:"project_id"`
	CredentialsFile string `yaml:"credentials_file"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {

		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.applyDefaults()

	return &cfg, nil
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	return cfg
}

func (c *Config) applyDefaults() {

	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}

	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}

	if c.Redis.Host == "" {
		c.Redis.Host = "localhost"
	}
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}

	if c.Neo4j.URI == "" {
		c.Neo4j.URI = "bolt://localhost:7687"
	}

	if c.Scanner.Workers == 0 {
		c.Scanner.Workers = 10
	}
	if c.Scanner.BatchSize == 0 {
		c.Scanner.BatchSize = 100
	}
	if c.Scanner.ScanTimeout == 0 {
		c.Scanner.ScanTimeout = 5 * time.Minute
	}
	if c.Scanner.MaxFileSize == 0 {
		c.Scanner.MaxFileSize = 100 * 1024 * 1024
	}
	if c.Scanner.SampleSize == 0 {
		c.Scanner.SampleSize = 1 * 1024 * 1024
	}
	if c.Scanner.FilesPerBucket == 0 {
		c.Scanner.FilesPerBucket = 1000
	}
	if c.Scanner.RandomSamplePct == 0 {
		c.Scanner.RandomSamplePct = 0.10
	}

	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "change-me-in-production"

		fmt.Println("WARNING: Using default JWT secret. Set auth.jwt_secret in production!")
	}
	if c.Auth.AccessTokenExpiry == 0 {
		c.Auth.AccessTokenExpiry = 15 * time.Minute
	}
	if c.Auth.RefreshTokenExpiry == 0 {
		c.Auth.RefreshTokenExpiry = 7 * 24 * time.Hour
	}

	if c.Notifications.MinSeverity == "" {
		c.Notifications.MinSeverity = models.SensitivityHigh
	}
	if c.Notifications.Email.SMTPPort == 0 {
		c.Notifications.Email.SMTPPort = 587
	}
}
