package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/controller"
	"github.com/AnarManafov/dataharbor/app/route"
	"github.com/AnarManafov/dataharbor/app/util"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize command-line flags first and check if we should continue
	// This handles --help and --version before any other initialization
	if !config.InitCmd() {
		// User requested help or version, exit cleanly
		return
	}

	// Initialize everything in the proper order
	initialize()
	stop := make(chan struct{})
	startServer(stop)
}

func initialize() {
	// 1. Load the config from the file path specified in the command line
	// (InitCmd has already been called in main())
	fmt.Printf("Loading config from: %s\n", config.ConfigFile)
	cfg, err := config.LoadConfig(config.ConfigFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Set the loaded config as the global config
	config.SetConfig(cfg)

	// 3. Initialize logger with the unified configuration
	common.InitLogger(&cfg.Logging)

	// 4. Load config into viper for components that still use viper directly (deprecated)
	err = config.LoadViper(config.ConfigFile)
	if err != nil {
		common.Logger.Warn("Warning: viper config loading issue:", err)
	}

	// 5. Log configuration information
	common.Logger.Info("Configuration loaded successfully")
	common.Logger.Info("Environment:", cfg.Env)
	common.Logger.Info("Server address:", cfg.Server.Address)
	common.Logger.Info("Auth enabled:", cfg.Auth.Enabled)

	if cfg.Auth.Enabled {
		common.Logger.Info("OIDC Issuer:", cfg.Auth.OIDC.Issuer)
		common.Logger.Info("OIDC ClientID:", cfg.Auth.OIDC.ClientID)
		clientSecretStatus := "not set"
		if cfg.Auth.OIDC.ClientSecret != "" {
			clientSecretStatus = "set"
		}
		common.Logger.Info("OIDC ClientSecret:", clientSecretStatus)
	}

	// 6. Log full configuration if debug is enabled
	if cfg.Server.Debug {
		common.Logger.Debug("Debug mode enabled")
		// Note: removed viper.AllSettings() as we're moving away from global viper
	}

	// 7. Initialize authentication system
	controller.InitAuth()

	// 8. Initialize snowflake ID generator
	if err := util.InitSnowflake(); err != nil {
		common.Logger.Warn("Failed to initialize snowflake ID generator:", err)
	}
}

func startServer(stop chan struct{}) {
	cfg := config.GetConfig()

	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	route.RegisterRoutes(r)

	// Use address from main config instead of separate port
	address := cfg.Server.Address
	if address == "" {
		address = ":8080" // Default fallback
	}

	srv := &http.Server{
		Addr:           address,
		Handler:        r,
		ReadTimeout:    0,                 // No read timeout for streaming downloads
		WriteTimeout:   0,                 // No write timeout for streaming downloads
		IdleTimeout:    120 * time.Second, // Keep connections alive
		MaxHeaderBytes: 1 << 20,           // 1MB max header size
	}

	// Start server with SSL/TLS support if enabled
	if cfg.Server.SSL.Enabled {
		fmt.Printf("Starting HTTPS server on address: %s\n", address)
		common.Logger.Infof("SSL enabled - cert: %s, key: %s", cfg.Server.SSL.CertFile, cfg.Server.SSL.KeyFile)

		// Validate certificate files exist
		if cfg.Server.SSL.CertFile == "" || cfg.Server.SSL.KeyFile == "" {
			common.Logger.Fatal("SSL enabled but certificate or key file not specified")
		}

		go func() {
			if err := srv.ListenAndServeTLS(cfg.Server.SSL.CertFile, cfg.Server.SSL.KeyFile); err != nil && err != http.ErrServerClosed {
				common.Logger.Fatal("HTTPS server failed:", err)
			}
		}()
	} else {
		fmt.Printf("Starting HTTP server on address: %s\n", address)
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				common.Logger.Fatal("HTTP server failed:", err)
			}
		}()
	}

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		common.Logger.Fatal("Server forced to shutdown:", err)
	}
}
