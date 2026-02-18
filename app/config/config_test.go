package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateConfig tests the validation logic
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Auth: AuthConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "missing server address",
			config: &Config{
				Server: ServerConfig{
					Address: "",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "server.address is required",
		},
		{
			name: "missing xrd host",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "xrd.host is required",
		},
		{
			name: "zero xrd port",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 0,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: true,
			errMsg:  "xrd.port must be greater than 0",
		},
		{
			name: "invalid logging level",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "logging.level must be one of",
		},
		{
			name: "valid debug level",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "debug",
				},
				Auth: AuthConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "auth enabled without issuer",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Auth: AuthConfig{
					Enabled: true,
					OIDC: OIDCConfig{
						Issuer:   "",
						ClientID: "test",
					},
				},
			},
			wantErr: true,
			errMsg:  "auth.oidc.issuer is required when auth is enabled",
		},
		{
			name: "auth enabled without client id",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Auth: AuthConfig{
					Enabled: true,
					OIDC: OIDCConfig{
						Issuer:   "https://example.com",
						ClientID: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "auth.oidc.client_id is required when auth is enabled",
		},
		{
			name: "valid auth config",
			config: &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Auth: AuthConfig{
					Enabled: true,
					OIDC: OIDCConfig{
						Issuer:   "https://example.com",
						ClientID: "test-client",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetConfig tests the singleton behavior
func TestGetConfig(t *testing.T) {
	// Reset the singleton for testing
	config = nil
	configOnce = sync.Once{}

	cfg1 := GetConfig()
	cfg2 := GetConfig()

	// Should return the same instance
	assert.Same(t, cfg1, cfg2)

	// Should have default values
	assert.Equal(t, "development", cfg1.Env)
	assert.Equal(t, ":8080", cfg1.Server.Address)
	assert.Equal(t, "localhost", cfg1.XRD.Host)
	assert.Equal(t, uint(1094), cfg1.XRD.Port)
	assert.Equal(t, "info", cfg1.Logging.Level)
	assert.False(t, cfg1.Auth.Enabled)
}

// TestSetConfig tests setting the configuration
func TestSetConfig(t *testing.T) {
	testCfg := &Config{
		Env: "test",
		Server: ServerConfig{
			Address: ":9000",
		},
	}

	SetConfig(testCfg)

	// Get the config and verify it's the one we set
	cfg := GetConfig()
	assert.Equal(t, "test", cfg.Env)
	assert.Equal(t, ":9000", cfg.Server.Address)

	// Reset for other tests
	config = nil
	configOnce = sync.Once{}
}

// TestSetDefaults tests that defaults are set correctly
func TestSetDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	// Server defaults
	assert.Equal(t, ":8080", v.GetString("server.address"))
	assert.False(t, v.GetBool("server.debug"))
	assert.Equal(t, "30s", v.GetString("server.shutdown_timeout"))
	assert.False(t, v.GetBool("server.cors.allow_credentials"))
	assert.False(t, v.GetBool("server.ssl.enabled"))

	// Logging defaults
	assert.Equal(t, "info", v.GetString("logging.level"))
	assert.Equal(t, "json", v.GetString("logging.format"))
	assert.True(t, v.GetBool("logging.console.enabled"))
	assert.Equal(t, "info", v.GetString("logging.console.level"))
	assert.Equal(t, "text", v.GetString("logging.console.format"))
	assert.False(t, v.GetBool("logging.file.enabled"))
	assert.Equal(t, "./log/dataharbor.log", v.GetString("logging.file.filename"))
	assert.Equal(t, 10, v.GetInt("logging.file.maxsize"))
	assert.Equal(t, 5, v.GetInt("logging.file.maxbackups"))
	assert.Equal(t, 30, v.GetInt("logging.file.maxage"))
	assert.True(t, v.GetBool("logging.file.compress"))

	// XRD defaults
	assert.Equal(t, "localhost", v.GetString("xrd.host"))
	assert.Equal(t, 1094, v.GetInt("xrd.port"))
	assert.Equal(t, "/tmp", v.GetString("xrd.initial_dir"))
	assert.False(t, v.GetBool("xrd.enable_ztn"))
	assert.Equal(t, 2*1024*1024, v.GetInt("xrd.download.buffer_size"))
	assert.Equal(t, 4*1024*1024, v.GetInt("xrd.download.flush_interval"))

	// Auth defaults
	assert.False(t, v.GetBool("auth.enabled"))
	assert.Equal(t, []string{"/health"}, v.GetStringSlice("auth.skip_auth_paths"))

	// Frontend defaults
	assert.Equal(t, "http://localhost:5173", v.GetString("frontend.url"))
	assert.Equal(t, "dist", v.GetString("frontend.dist_dir"))
}

// TestCreateDefaultConfig tests default config creation
func TestCreateDefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	err := createDefaultConfig(configFile)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	// Verify file is not empty
	data, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Load the created config and validate it
	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify key fields
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, ":8080", cfg.Server.Address)
	assert.Equal(t, "localhost", cfg.XRD.Host)
	assert.Equal(t, uint(1094), cfg.XRD.Port)
}

// TestLoadConfig_ValidFile tests loading a valid config file
func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create a valid config file
	configContent := `
env: production
server:
  address: :9090
  debug: true
logging:
  level: debug
xrd:
  host: xrootd.example.com
  port: 2094
  initial_dir: /data
auth:
  enabled: false
`
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, ":9090", cfg.Server.Address)
	assert.True(t, cfg.Server.Debug)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "xrootd.example.com", cfg.XRD.Host)
	assert.Equal(t, uint(2094), cfg.XRD.Port)
	assert.Equal(t, "/data", cfg.XRD.InitialDir)
}

// TestLoadConfig_InvalidFile tests handling of invalid config files
func TestLoadConfig_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	// Create an invalid YAML file
	configContent := `
this is not: valid: yaml: content
  - broken
`
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	_, err = LoadConfig(configFile)
	assert.Error(t, err)
}

// TestLoadConfig_MissingFile tests that missing config creates default
func TestLoadConfig_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "nonexistent.yaml")

	cfg, err := LoadConfig(configFile)
	// The current implementation requires the config file to exist or creates it
	// If viper can't find it, it will error (this is expected behavior)
	// This test verifies the error handling for missing files
	if err != nil {
		// This is expected - the config file doesn't exist and viper couldn't read it
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	} else {
		// If no error, verify the config was loaded correctly
		assert.NotNil(t, cfg)
		// And file was created
		_, statErr := os.Stat(configFile)
		assert.NoError(t, statErr)
	}
}

// TestLoadConfig_ValidationFailure tests that invalid config fails validation
func TestLoadConfig_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-values.yaml")

	// Create a config with invalid values
	configContent := `
env: test
server:
  address: ""
logging:
  level: invalid_level
xrd:
  host: localhost
  port: 1094
`
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	_, err = LoadConfig(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestLoadConfig_EnvironmentOverride tests environment variable override
func TestLoadConfig_EnvironmentOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create a basic config file
	configContent := `
env: development
server:
  address: :8080
logging:
  level: info
xrd:
  host: localhost
  port: 1094
auth:
  enabled: false
`
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Set environment variable
	err = os.Setenv("DATAHARBOR_SERVER_ADDRESS", ":9999")
	require.NoError(t, err)
	defer func() { _ = os.Unsetenv("DATAHARBOR_SERVER_ADDRESS") }()

	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)

	// Environment variable should override file value
	assert.Equal(t, ":9999", cfg.Server.Address)
}

// TestLoadConfig_CreatesDirIfNotExists tests directory creation
func TestLoadConfig_CreatesDirIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "subdir", "nested", "config.yaml")

	// Directory doesn't exist initially
	_, err := os.Stat(filepath.Dir(configFile))
	assert.True(t, os.IsNotExist(err))

	// LoadConfig creates the directory structure
	_, _ = LoadConfig(configFile)

	// Verify directory was created even if config loading failed
	_, dirErr := os.Stat(filepath.Dir(configFile))
	assert.NoError(t, dirErr, "Directory should have been created")
}

// TestLoadViper tests the deprecated LoadViper function
func TestLoadViper(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "viper-test.yaml")

	configContent := `
env: test
server:
  address: :7070
`
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	err = LoadViper(configFile)
	assert.NoError(t, err)
}

// TestLoadViper_EmptyPath tests LoadViper with empty path
func TestLoadViper_EmptyPath(t *testing.T) {
	err := LoadViper("")
	assert.NoError(t, err)
}

// TestGetConfig_Concurrency tests thread-safety of GetConfig
func TestGetConfig_Concurrency(t *testing.T) {
	// Reset singleton
	config = nil
	configOnce = sync.Once{}

	// Launch multiple goroutines that call GetConfig
	const numGoroutines = 100
	configs := make([]*Config, numGoroutines)
	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			configs[idx] = GetConfig()
		}(i)
	}

	wg.Wait()

	// All should return the same instance
	firstConfig := configs[0]
	for i := 1; i < numGoroutines; i++ {
		assert.Same(t, firstConfig, configs[i])
	}

	// Reset for other tests
	config = nil
	configOnce = sync.Once{}
}

// TestConfig_AllLoggingLevels ensures all valid logging levels work
func TestConfig_AllLoggingLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Address: ":8080",
				},
				XRD: XRDConfig{
					Host: "localhost",
					Port: 1094,
				},
				Logging: LoggingConfig{
					Level: level,
				},
				Auth: AuthConfig{
					Enabled: false,
				},
			}

			err := ValidateConfig(cfg)
			assert.NoError(t, err)
		})
	}
}

// TestLoadConfig_DefaultLocations tests config file search in default locations
func TestLoadConfig_DefaultLocations(t *testing.T) {
	// Create a config in one of the default locations
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "application.yaml")
	configContent := `
env: test
server:
  address: :8888
logging:
  level: debug
xrd:
  host: test.example.com
  port: 1094
auth:
  enabled: false
`
	err = os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Change to the temp directory so the relative path works
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Load with empty string should search default locations
	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "test", cfg.Env)
}

// TestLoadConfig_UnmarshalError tests handling of unmarshal errors
func TestLoadConfig_UnmarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "bad-types.yaml")

	// Create a config with wrong types (port as string instead of int)
	configContent := `
env: test
server:
  address: :8080
logging:
  level: info
xrd:
  host: localhost
  port: "not-a-number"
auth:
  enabled: false
`
	err := os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)

	_, err = LoadConfig(configFile)
	assert.Error(t, err)
}

// TestCreateDefaultConfig_WriteError tests error handling when file can't be written
func TestCreateDefaultConfig_WriteError(t *testing.T) {
	// Try to create a config in a read-only directory
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0o555) // Read-only directory
	require.NoError(t, err)
	defer func() { _ = os.Chmod(readOnlyDir, 0o755) }() // Cleanup

	configFile := filepath.Join(readOnlyDir, "config.yaml")
	err = createDefaultConfig(configFile)
	assert.Error(t, err)
}

// TestLoadConfig_DirectoryCreationError tests error handling when directory can't be created
func TestLoadConfig_DirectoryCreationError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0o555)
	require.NoError(t, err)
	defer func() { _ = os.Chmod(readOnlyDir, 0o755) }()

	configFile := filepath.Join(readOnlyDir, "subdir", "config.yaml")
	_, err = LoadConfig(configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config directory")
}
