package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/core"
	"github.com/AnarManafov/dataharbor/app/route"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// Initialize everything in the proper order
	initialize()
	stop := make(chan struct{})
	startServer(stop)
}

func initialize() {
	// 1. Initialize command-line flags first to get the config file path
	config.InitCmd()

	// 2. Initialize logger
	common.InitLogger()

	// 3. Load the config from the file path specified in the command line
	common.Logger.Info("Loading config from:", config.ConfigFile)
	cfg, err := config.LoadConfig(config.ConfigFile)
	if err != nil {
		common.Logger.Error("Failed to load config:", err)
	} else {
		// Set the loaded config as the global config
		config.SetConfig(cfg)

		// Log key configuration values for debugging
		common.Logger.Info("Auth configuration - Enabled:", cfg.Auth.Enabled)
		common.Logger.Info("OIDC configuration - Issuer:", cfg.Auth.OIDC.Issuer)
		common.Logger.Info("OIDC configuration - ClientID:", cfg.Auth.OIDC.ClientID)
		clientSecretStatus := "not set"
		if cfg.Auth.OIDC.ClientSecret != "" {
			clientSecretStatus = "set"
		}
		common.Logger.Info("OIDC configuration - ClientSecret:", clientSecretStatus)
	}

	// 4. Load config into viper for components that use viper directly
	err = config.LoadViper(config.ConfigFile)
	if err != nil {
		common.Logger.Warn("Warning: viper config loading issue:", err)
	}

	// 5. Log full configuration if debug is enabled
	if cfg.Server.Debug {
		common.Logger.Debug("Full configuration:", viper.AllSettings())
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

	ticker, done := core.NewSanitationScheduler()
	go core.SanitationJob(ticker, done)

	srv := &http.Server{
		Addr:    address,
		Handler: r,
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

	ticker.Stop()
	done <- true
}
