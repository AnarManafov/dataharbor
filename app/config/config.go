package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ConfigFile is the path to the configuration file
var ConfigFile string

var (
	config     *Config
	configOnce sync.Once
)

// Config represents the application configuration
type Config struct {
	Env      string         `mapstructure:"env" yaml:"env"`
	Server   ServerConfig   `mapstructure:"server" yaml:"server"`
	Logging  LoggingConfig  `mapstructure:"logging" yaml:"logging"`
	XRD      XRDConfig      `mapstructure:"xrd" yaml:"xrd"`
	Auth     AuthConfig     `mapstructure:"auth" yaml:"auth"`
	Frontend FrontendConfig `mapstructure:"frontend" yaml:"frontend"`
}

// FrontendConfig represents the frontend configuration
type FrontendConfig struct {
	URL        string   `mapstructure:"url" yaml:"url"`
	AssetPaths []string `mapstructure:"asset_paths" yaml:"asset_paths"`
	DistDir    string   `mapstructure:"dist_dir" yaml:"dist_dir"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Address         string     `mapstructure:"address" yaml:"address"`
	Debug           bool       `mapstructure:"debug" yaml:"debug"`
	ShutdownTimeout string     `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout"`
	CORS            CORSConfig `mapstructure:"cors" yaml:"cors"`
	SSL             SSLConfig  `mapstructure:"ssl" yaml:"ssl"`
}

// CORSConfig represents the CORS configuration
type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins" yaml:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods" yaml:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers" yaml:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials" yaml:"allow_credentials"`
}

// SSLConfig represents the SSL configuration
type SSLConfig struct {
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled"`
	CertFile string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile  string `mapstructure:"key_file" yaml:"key_file"`
}

// LoggingConfig represents the unified logging configuration
type LoggingConfig struct {
	Level   string        `mapstructure:"level" yaml:"level"`
	Format  string        `mapstructure:"format" yaml:"format"`
	Console ConsoleConfig `mapstructure:"console" yaml:"console"`
	File    FileConfig    `mapstructure:"file" yaml:"file"`
}

// ConsoleConfig represents console logging configuration
type ConsoleConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
	Level   string `mapstructure:"level" yaml:"level"`
	Format  string `mapstructure:"format" yaml:"format"`
}

// FileConfig represents file logging configuration with rotation
type FileConfig struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	Level      string `mapstructure:"level" yaml:"level"`
	Format     string `mapstructure:"format" yaml:"format"`
	Filename   string `mapstructure:"filename" yaml:"filename"`
	MaxSize    int    `mapstructure:"maxsize" yaml:"maxsize"`       // MB
	MaxBackups int    `mapstructure:"maxbackups" yaml:"maxbackups"` // Number of backups
	MaxAge     int    `mapstructure:"maxage" yaml:"maxage"`         // Days
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
}

// XRDConfig represents the XRootD configuration
type XRDConfig struct {
	Host       string         `mapstructure:"host" yaml:"host"`
	Port       uint           `mapstructure:"port" yaml:"port"`
	InitialDir string         `mapstructure:"initial_dir" yaml:"initial_dir"`
	User       string         `mapstructure:"user" yaml:"user"`
	UserGroup  string         `mapstructure:"usergroup" yaml:"usergroup"`
	EnableZTN  bool           `mapstructure:"enable_ztn" yaml:"enable_ztn"` // Enable ZTN protocol (TLS + OAuth token authentication)
	ClientCert string         `mapstructure:"client_cert" yaml:"client_cert"`
	ClientKey  string         `mapstructure:"client_key" yaml:"client_key"`
	Download   DownloadConfig `mapstructure:"download" yaml:"download"`
}

// DownloadConfig represents file download optimization settings
type DownloadConfig struct {
	// BufferSize is the size of the buffer used for streaming file downloads (in bytes)
	// Larger buffers reduce protocol overhead and improve throughput for large files
	// Recommended: 2MB (2097152) for multi-GB scientific data transfers over WAN
	// Trade-off: Memory usage = BufferSize × concurrent downloads
	BufferSize int `mapstructure:"buffer_size" yaml:"buffer_size"`

	// FlushInterval controls how often the response buffer is flushed to the client (in bytes)
	// Balances responsiveness with performance - smaller values provide more frequent progress updates
	FlushInterval int `mapstructure:"flush_interval" yaml:"flush_interval"`
}

// AuthConfig represents the authentication configuration
type AuthConfig struct {
	Enabled       bool       `mapstructure:"enabled" yaml:"enabled"`
	SkipAuthPaths []string   `mapstructure:"skip_auth_paths" yaml:"skip_auth_paths"`
	OIDC          OIDCConfig `mapstructure:"oidc" yaml:"oidc"`
}

// OIDCConfig represents the OIDC configuration
type OIDCConfig struct {
	Issuer                string   `mapstructure:"issuer" yaml:"issuer"`
	ClientID              string   `mapstructure:"client_id" yaml:"client_id"`
	ClientSecret          string   `mapstructure:"client_secret" yaml:"client_secret"`
	DiscoveryURL          string   `mapstructure:"discovery_url" yaml:"discovery_url"`
	AllowedRoles          []string `mapstructure:"allowed_roles" yaml:"allowed_roles"`
	SessionSecret         string   `mapstructure:"session_secret" yaml:"session_secret"`
	TokenRefreshBufferSec int64    `mapstructure:"token_refresh_buffer_sec" yaml:"token_refresh_buffer_sec"`
}

// ValidateConfig validates critical configuration fields
func ValidateConfig(cfg *Config) error {
	// Validate server configuration
	if cfg.Server.Address == "" {
		return fmt.Errorf("server.address is required")
	}

	// Validate XRD configuration
	if cfg.XRD.Host == "" {
		return fmt.Errorf("xrd.host is required")
	}
	if cfg.XRD.Port == 0 {
		return fmt.Errorf("xrd.port must be greater than 0")
	}

	// Validate auth configuration
	if cfg.Auth.Enabled {
		if cfg.Auth.OIDC.Issuer == "" {
			return fmt.Errorf("auth.oidc.issuer is required when auth is enabled")
		}
		if cfg.Auth.OIDC.ClientID == "" {
			return fmt.Errorf("auth.oidc.client_id is required when auth is enabled")
		}
	}

	// Validate logging configuration
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := slices.Contains(validLevels, cfg.Logging.Level)
	if !levelValid {
		return fmt.Errorf("logging.level must be one of: %v", validLevels)
	}

	return nil
}

// LoadConfig loads the configuration from file using Viper
func LoadConfig(configFile string) (*Config, error) {
	// Initialize Viper
	v := viper.New()

	// Set environment variable support
	v.SetEnvPrefix("DATAHARBOR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set default values
	setDefaults(v)

	// If configFile is not provided, use default paths
	if configFile == "" {
		// Try to locate the config file in common locations
		locations := []string{
			"config/application.yaml",
			"application.yaml",
			"../config/application.yaml",
			"/etc/dataharbor/config/application.yaml",
		}

		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				configFile = loc
				break
			}
		}

		// If still not found, use the first one
		if configFile == "" {
			configFile = locations[0]
		}
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configFile)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Set config file
	v.SetConfigFile(configFile)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// If config file doesn't exist, create a default one
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := createDefaultConfig(configFile); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			// Try to read again
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config after creating default: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Set the loaded config as the global config
	config = &cfg

	return &cfg, nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	configOnce.Do(func() {
		if config == nil {
			config = &Config{
				Env: "development",
				Server: ServerConfig{
					Address: ":8080",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "text",
					Console: ConsoleConfig{
						Enabled: true,
						Level:   "info",
						Format:  "text",
					},
					File: FileConfig{
						Enabled:    false,
						Level:      "info",
						Format:     "json",
						Filename:   "./log/dataharbor.log",
						MaxSize:    10,
						MaxBackups: 5,
						MaxAge:     30,
						Compress:   true,
					},
				},
				XRD: XRDConfig{
					Host:       "localhost",
					Port:       1094,
					InitialDir: "/tmp",
					EnableZTN:  false,
					Download: DownloadConfig{
						BufferSize:    2 * 1024 * 1024, // 2MB
						FlushInterval: 4 * 1024 * 1024, // 4MB
					},
				},
				Auth: AuthConfig{
					Enabled: false,
					SkipAuthPaths: []string{
						"/health",
					},
					OIDC: OIDCConfig{
						TokenRefreshBufferSec: 60, // Default: refresh tokens 1 minute before expiration
					},
				},
				Frontend: FrontendConfig{
					URL:        "http://localhost:5173",
					AssetPaths: []string{},
					DistDir:    "dist",
				},
			}
		}
	})
	return config
}

// SetConfig sets the current configuration
func SetConfig(cfg *Config) {
	config = cfg
}

// LoadViper loads configuration into Viper (deprecated - use LoadConfig instead)
// This function is kept for backward compatibility with existing code
func LoadViper(configFile string) error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		viper.SetEnvPrefix("DATAHARBOR")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()
		return viper.ReadInConfig()
	}

	// Set default values for components that still use global viper
	setDefaults(viper.GetViper())
	return nil
}

// setDefaults sets default values for Viper configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.debug", false)
	v.SetDefault("server.shutdown_timeout", "30s")
	v.SetDefault("server.cors.allow_credentials", false)
	v.SetDefault("server.ssl.enabled", false)

	// Logging defaults (optimized values)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.console.enabled", true)
	v.SetDefault("logging.console.level", "info")
	v.SetDefault("logging.console.format", "text")
	v.SetDefault("logging.file.enabled", false)
	v.SetDefault("logging.file.level", "info")
	v.SetDefault("logging.file.format", "json")
	v.SetDefault("logging.file.filename", "./log/dataharbor.log")
	v.SetDefault("logging.file.maxsize", 10)
	v.SetDefault("logging.file.maxbackups", 5)
	v.SetDefault("logging.file.maxage", 30)
	v.SetDefault("logging.file.compress", true)

	// XRD defaults
	v.SetDefault("xrd.host", "localhost")
	v.SetDefault("xrd.port", 1094)
	v.SetDefault("xrd.initial_dir", "/tmp")
	v.SetDefault("xrd.enable_ztn", false)
	v.SetDefault("xrd.download.buffer_size", 2*1024*1024)    // 2MB - optimal for multi-GB files over WAN
	v.SetDefault("xrd.download.flush_interval", 4*1024*1024) // 4MB - balance between responsiveness and performance

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.skip_auth_paths", []string{"/health"})

	// Frontend defaults
	v.SetDefault("frontend.url", "http://localhost:5173")
	v.SetDefault("frontend.asset_paths", []string{})
	v.SetDefault("frontend.dist_dir", "dist")
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configFile string) error {
	defaultConfig := &Config{
		Env: "development",
		Server: ServerConfig{
			Address:         ":8080",
			Debug:           false,
			ShutdownTimeout: "30s",
			CORS: CORSConfig{
				AllowCredentials: false,
				AllowOrigins:     []string{},
				AllowMethods:     []string{},
				AllowHeaders:     []string{},
			},
			SSL: SSLConfig{
				Enabled:  false,
				CertFile: "",
				KeyFile:  "",
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Console: ConsoleConfig{
				Enabled: true,
				Level:   "info",
				Format:  "text",
			},
			File: FileConfig{
				Enabled:    false,
				Level:      "info",
				Format:     "json",
				Filename:   "./log/dataharbor.log",
				MaxSize:    10,
				MaxBackups: 5,
				MaxAge:     30,
				Compress:   true,
			},
		},
		XRD: XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp",
			User:       "",
			UserGroup:  "",
			EnableZTN:  false,
			ClientCert: "",
			ClientKey:  "",
			Download: DownloadConfig{
				BufferSize:    2 * 1024 * 1024, // 2MB - optimal for multi-GB files
				FlushInterval: 4 * 1024 * 1024, // 4MB - balance responsiveness and performance
			},
		},
		Auth: AuthConfig{
			Enabled: false,
			SkipAuthPaths: []string{
				"/health",
			},
			OIDC: OIDCConfig{
				Issuer:                "",
				ClientID:              "",
				ClientSecret:          "",
				DiscoveryURL:          "",
				AllowedRoles:          []string{},
				SessionSecret:         "",
				TokenRefreshBufferSec: 60, // Default: refresh tokens 1 minute before expiration
			},
		},
		Frontend: FrontendConfig{
			URL:        "http://localhost:5173",
			AssetPaths: []string{},
			DistDir:    "dist",
		},
	}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}

	return nil
}
