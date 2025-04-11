package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
	"github.com/AnarManafov/data_lake_ui/app/core"
	"github.com/AnarManafov/data_lake_ui/app/route"

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

	// 5. Parse XRD config from viper values
	common.ParseXrdConfig()

	// 6. Parse system config
	common.ParseSystemConfig()

	// 7. Log full configuration if debug is enabled
	if viper.GetBool("server.debug") {
		common.Logger.Debug("Full configuration:", viper.AllSettings())
	}
}

func startServer(stop chan struct{}) {
	if !common.ServerConfig.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	route.RegisterRoutes(r)

	port := strconv.Itoa(common.ServerConfig.Port)
	fmt.Printf("Starting server on port: %s\n", port)

	ticker, done := core.NewSanitationScheduler()
	go core.SanitationJob(ticker, done)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			common.Logger.Fatal(err)
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		common.Logger.Fatal("Server forced to shutdown:", err)
	}

	ticker.Stop()
	done <- true
}
