package config

import (
	"os"
	"path/filepath"
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
	Env      string         `yaml:"env"`
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
	XRD      XRDConfig      `yaml:"xrd"`
	Auth     AuthConfig     `yaml:"auth"`
	Frontend FrontendConfig `yaml:"frontend"` // Added Frontend config
}

// FrontendConfig represents the frontend configuration
type FrontendConfig struct {
	URL        string   `yaml:"url"`         // URL where the frontend is hosted
	AssetPaths []string `yaml:"asset_paths"` // Search paths for frontend assets, in order of priority
	DistDir    string   `yaml:"dist_dir"`    // Distribution directory name (default: "dist")
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Address         string     `yaml:"address"`
	ShutdownTimeout string     `yaml:"shutdown_timeout"`
	CORS            CORSConfig `yaml:"cors"`
	SSL             SSLConfig  `yaml:"ssl"`
}

// CORSConfig represents the CORS configuration
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

// SSLConfig represents the SSL configuration
type SSLConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// LogConfig represents the logging configuration
type LogConfig struct {
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
}

// XRDConfig represents the XRootD configuration
type XRDConfig struct {
	URL          string `yaml:"url"`
	User         string `yaml:"user"`
	UserGroup    string `yaml:"usergroup"`
	UserPwd      string `yaml:"userpwd"`
	UserRequired bool   `yaml:"user_required"`
	TLS          bool   `yaml:"tls"`
	ClientCert   string `yaml:"client_cert"`
	ClientKey    string `yaml:"client_key"`
}

// AuthConfig represents the authentication configuration
type AuthConfig struct {
	Enabled       bool       `yaml:"enabled"`
	SkipAuthPaths []string   `yaml:"skip_auth_paths"`
	OIDC          OIDCConfig `yaml:"oidc"`
}

// OIDCConfig represents the OIDC configuration
type OIDCConfig struct {
	Issuer        string   `yaml:"issuer"`
	ClientID      string   `yaml:"client_id"`
	ClientSecret  string   `yaml:"client_secret"`
	DiscoveryURL  string   `yaml:"discovery_url"`
	AllowedRoles  []string `yaml:"allowed_roles"`
	SessionSecret string   `yaml:"session_secret"`
}

// LoadConfig loads the configuration from file
func LoadConfig(configFile string) (*Config, error) {
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
			return nil, err
		}
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// If not, look for a template
		templateFile := configFile + ".template"
		if _, err := os.Stat(templateFile); err == nil {
			// Copy template to config file
			templateData, err := os.ReadFile(templateFile)
			if err != nil {
				return nil, err
			}
			// Fix: corrected syntax for WriteFile call
			err = os.WriteFile(configFile, templateData, 0644)
			if err != nil {
				return nil, err
			}
		} else {
			// Create a minimal config file
			defaultConfig := &Config{
				Env: "development",
				Server: ServerConfig{
					Address: ":8080",
				},
				Log: LogConfig{
					Format: "text",
					Level:  "info",
				},
				XRD: XRDConfig{
					URL:          "root://localhost:1094",
					UserRequired: false,
				},
				Auth: AuthConfig{
					Enabled: false,
					SkipAuthPaths: []string{
						"/health",
					},
				},
				Frontend: FrontendConfig{
					URL:        "http://localhost:5173", // Default frontend URL for development
					AssetPaths: []string{},
					DistDir:    "dist",
				},
			}
			data, err := yaml.Marshal(defaultConfig)
			if err != nil {
				return nil, err
			}
			// Fix: corrected syntax for WriteFile call
			err = os.WriteFile(configFile, data, 0644)
			if err != nil {
				return nil, err
			}
		}
	}

	// Read and parse config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Set the loaded config as the global config
	config = cfg

	return cfg, nil
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
				Log: LogConfig{
					Format: "text",
					Level:  "info",
				},
				XRD: XRDConfig{
					URL: "root://localhost:1094",
				},
				Auth: AuthConfig{
					Enabled: false,
					SkipAuthPaths: []string{
						"/health",
					},
				},
				Frontend: FrontendConfig{
					URL:        "http://localhost:5173", // Default frontend URL for development
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

// LoadViper loads configuration into Viper
func LoadViper(configFile string) error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		return viper.ReadInConfig()
	}

	// Set default values for viper
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.debug", false)
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.level", "info")

	// Defaults for XRD
	viper.SetDefault("xrd.host", "localhost")
	viper.SetDefault("xrd.port", 1094)
	viper.SetDefault("xrd.initial_dir", "/tmp")
	viper.SetDefault("xrd.xrd_client_bin_path", "/opt/homebrew/bin/")
	viper.SetDefault("xrd.process_timeout", 60)
	viper.SetDefault("xrd.staging_path", "/tmp/delete_me")
	viper.SetDefault("xrd.sanitation_job_interval", 30)
	viper.SetDefault("xrd.staging_tmp_dir_prefix", "stg_")
	viper.SetDefault("xrd.url", "root://localhost:1094")
	viper.SetDefault("xrd.user_required", false)

	// OIDC defaults
	viper.SetDefault("auth.enabled", false)
	viper.SetDefault("auth.skip_auth_paths", []string{"/health"})

	// Frontend defaults
	viper.SetDefault("frontend.url", "http://localhost:5173")
	viper.SetDefault("frontend.asset_paths", []string{})
	viper.SetDefault("frontend.dist_dir", "dist")

	return nil
}
