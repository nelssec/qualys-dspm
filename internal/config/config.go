package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/qualys/dspm/internal/models"
)

// Config holds all application configuration
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

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret          string        `yaml:"jwt_secret"`
	AccessTokenExpiry  time.Duration `yaml:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `yaml:"refresh_token_expiry"`
}

// NotificationsConfig holds notification configuration
type NotificationsConfig struct {
	MinSeverity models.Sensitivity `yaml:"min_severity"`
	Slack       SlackNotifyConfig  `yaml:"slack"`
	Email       EmailNotifyConfig  `yaml:"email"`
}

// SlackNotifyConfig holds Slack notification settings
type SlackNotifyConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

// EmailNotifyConfig holds email notification settings
type EmailNotifyConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
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

// DSN returns the PostgreSQL connection string
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// Addr returns the Redis address
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Neo4jConfig holds Neo4j configuration
type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// ScannerConfig holds scanner worker configuration
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

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Region          string `yaml:"region"`
	AssumeRoleARN   string `yaml:"assume_role_arn"`
	ExternalID      string `yaml:"external_id"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

// AzureConfig holds Azure-specific configuration
type AzureConfig struct {
	TenantID       string `yaml:"tenant_id"`
	ClientID       string `yaml:"client_id"`
	ClientSecret   string `yaml:"client_secret"`
	SubscriptionID string `yaml:"subscription_id"`
}

// GCPConfig holds GCP-specific configuration
type GCPConfig struct {
	ProjectID       string `yaml:"project_id"`
	CredentialsFile string `yaml:"credentials_file"`
}

// Load reads and parses configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return default config if file doesn't exist
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Apply defaults for unset values
	cfg.applyDefaults()

	return &cfg, nil
}

// defaultConfig returns a configuration with sensible defaults
func defaultConfig() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	return cfg
}

func (c *Config) applyDefaults() {
	// Server defaults
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

	// Database defaults
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

	// Redis defaults
	if c.Redis.Host == "" {
		c.Redis.Host = "localhost"
	}
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}

	// Neo4j defaults
	if c.Neo4j.URI == "" {
		c.Neo4j.URI = "bolt://localhost:7687"
	}

	// Scanner defaults
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
		c.Scanner.MaxFileSize = 100 * 1024 * 1024 // 100MB
	}
	if c.Scanner.SampleSize == 0 {
		c.Scanner.SampleSize = 1 * 1024 * 1024 // 1MB
	}
	if c.Scanner.FilesPerBucket == 0 {
		c.Scanner.FilesPerBucket = 1000
	}
	if c.Scanner.RandomSamplePct == 0 {
		c.Scanner.RandomSamplePct = 0.10
	}

	// Auth defaults
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "change-me-in-production"
	}
	if c.Auth.AccessTokenExpiry == 0 {
		c.Auth.AccessTokenExpiry = 15 * time.Minute
	}
	if c.Auth.RefreshTokenExpiry == 0 {
		c.Auth.RefreshTokenExpiry = 7 * 24 * time.Hour
	}

	// Notifications defaults
	if c.Notifications.MinSeverity == "" {
		c.Notifications.MinSeverity = models.SensitivityHigh
	}
	if c.Notifications.Email.SMTPPort == 0 {
		c.Notifications.Email.SMTPPort = 587
	}
}
