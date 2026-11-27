package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
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

	ctx, stop := shutdownSignalContext()
	err := startServer(ctx)
	stop()
	if err != nil {
		// Fatal exits with status 1; a clean shutdown falls through and exits 0.
		common.Logger.Fatal("Server error: ", err)
	}
}

// defaultShutdownTimeout applies when server.shutdown_timeout is unset or invalid.
const defaultShutdownTimeout = 30 * time.Second

// shutdownSignalContext returns a context that is cancelled by the first
// SIGINT or SIGTERM. After that first signal the default handling is restored,
// so a second signal ends the process at once instead of waiting for the drain.
func shutdownSignalContext() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()
	return ctx, stop
}

// parseShutdownTimeout turns server.shutdown_timeout into a duration. The
// second result is false when the value is empty, malformed or not positive and
// the default was used instead.
func parseShutdownTimeout(value string) (time.Duration, bool) {
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return defaultShutdownTimeout, false
	}
	return d, true
}

// inFlightTracker counts the requests currently inside the handler, so a
// shutdown can report how many it drained and how many it had to cut.
type inFlightTracker struct {
	next   http.Handler
	active atomic.Int64
}

func (t *inFlightTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.active.Add(1)
	defer t.active.Add(-1)
	t.next.ServeHTTP(w, r)
}

// serverOptions is what serve needs beyond the listener and the handler.
type serverOptions struct {
	certFile        string // with keyFile: serve TLS
	keyFile         string
	shutdownTimeout time.Duration
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
		common.Logger.Info("OIDC client secret is ", clientSecretStatus)
	}

	// 6. Log full configuration if debug is enabled
	if cfg.Server.Debug {
		common.Logger.Debug("Debug mode enabled")
		// Note: removed viper.AllSettings() as we're moving away from global viper
	}

	// 7. Initialize authentication system
	controller.InitAuth()
	// Surface a wrong client secret at boot instead of as a 500 mid-login.
	// cfg and the logger are captured here, before the goroutine starts, so it
	// never reads the (re-assignable) globals concurrently.
	go controller.VerifyOIDCClientCredentials(cfg, common.GetLogger())

	// 8. Initialize snowflake ID generator
	if err := util.InitSnowflake(); err != nil {
		common.Logger.Warn("Failed to initialize snowflake ID generator:", err)
	}
}

// startServer builds the router, binds the configured address and serves until
// ctx is cancelled, then shuts down gracefully. It returns nil after a clean
// shutdown.
func startServer(ctx context.Context) error {
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

	timeout, ok := parseShutdownTimeout(cfg.Server.ShutdownTimeout)
	if !ok {
		common.GetLogger().Warnw("Invalid server.shutdown_timeout, using the default",
			"value", cfg.Server.ShutdownTimeout, "default", defaultShutdownTimeout.String())
	}
	opts := serverOptions{shutdownTimeout: timeout}

	scheme := "HTTP"
	if cfg.Server.SSL.Enabled {
		scheme = "HTTPS"
		common.GetLogger().Infof("SSL enabled - cert: %s, key: %s", cfg.Server.SSL.CertFile, cfg.Server.SSL.KeyFile)

		// Validate certificate files exist
		if cfg.Server.SSL.CertFile == "" || cfg.Server.SSL.KeyFile == "" {
			return errors.New("SSL enabled but certificate or key file not specified")
		}
		opts.certFile = cfg.Server.SSL.CertFile
		opts.keyFile = cfg.Server.SSL.KeyFile
	}

	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("%s server failed to listen on %s: %w", scheme, address, err)
	}
	fmt.Printf("Starting %s server on address: %s\n", scheme, address)

	return serve(ctx, ln, r, opts)
}

// serve runs handler on ln until ctx is cancelled or the listener fails. On
// cancellation it stops accepting connections, waits up to
// opts.shutdownTimeout for the requests in flight and for their responses to
// reach the clients, and then stops the upload janitor. A drain that does not
// finish in time is cut and reported as an error.
func serve(ctx context.Context, ln net.Listener, handler http.Handler, opts serverOptions) error {
	logger := common.GetLogger()
	tracker := &inFlightTracker{next: handler}
	dl := &drainListener{Listener: ln}

	srv := &http.Server{
		Handler:        tracker,
		ReadTimeout:    0,                 // No read timeout for streaming downloads
		WriteTimeout:   0,                 // No write timeout for streaming downloads
		IdleTimeout:    120 * time.Second, // Keep connections alive
		MaxHeaderBytes: 1 << 20,           // 1MB max header size
	}

	// A listener that dies on its own is reported through this channel rather
	// than by calling Logger.Fatal (os.Exit) from inside the goroutine: an exit
	// there takes down the whole process, which in tests means the test binary.
	serveErr := make(chan error, 1)
	go func() {
		var err error
		if opts.certFile != "" || opts.keyFile != "" {
			err = srv.ServeTLS(dl, opts.certFile, opts.keyFile)
		} else {
			err = srv.Serve(dl)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("server failed: %w", err)
		}
	}()

	// Whichever comes first: the listener failing, or a shutdown request.
	select {
	case err := <-serveErr:
		_ = ln.Close() // ServeTLS leaves it open when the certificate does not load
		return err
	case <-ctx.Done():
	}

	started := time.Now()
	logger.Infow("Shutdown started: no new connections are accepted",
		"timeout", opts.shutdownTimeout.String(), "inFlight", tracker.active.Load())

	drainCtx, cancel := context.WithTimeout(context.Background(), opts.shutdownTimeout)
	defer cancel()
	// From here on a connection is closed only once its response has reached the
	// client, or at the deadline: see drainListener.
	deadline, _ := drainCtx.Deadline()
	dl.startDrain(deadline)

	shutdownErr := srv.Shutdown(drainCtx)
	remaining := tracker.active.Load()
	if shutdownErr != nil {
		_ = srv.Close() // cut what is left
	}
	// Connections closed before or during the shutdown may still be delivering
	// the tail of a response; they give up at the deadline.
	dl.waitClosed()
	undelivered := dl.unsent.Load()
	janitorStopped := controller.StopUploadJanitor(drainCtx)

	if shutdownErr != nil || undelivered > 0 {
		logger.Errorw("Shutdown deadline reached: remaining requests were cut",
			"timeout", opts.shutdownTimeout.String(), "inFlight", remaining, "undelivered", undelivered)
		return fmt.Errorf("shutdown timed out after %s: %d request(s) still in flight, %d response(s) not fully delivered",
			opts.shutdownTimeout, remaining, undelivered)
	}

	if !janitorStopped {
		logger.Warnw("Upload janitor did not stop before the shutdown deadline")
	}
	logger.Infow("Shutdown complete: all requests drained",
		"duration", time.Since(started).Round(time.Millisecond).String())
	return nil
}
