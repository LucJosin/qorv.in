package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lucjosin/qorv.in/internal/api"
	"github.com/lucjosin/qorv.in/internal/database/mariadb"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
	"github.com/lucjosin/qorv.in/internal/domain/domain"
	"github.com/lucjosin/qorv.in/internal/domain/system"
	"github.com/lucjosin/qorv.in/internal/domain/user"
	"github.com/lucjosin/qorv.in/internal/domain/workspace"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/slogx"

	apiv1 "github.com/lucjosin/qorv.in/internal/api/v1"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type AppConfig struct {
	Domains       []string `env:"DOMAINS,required"`
	Username      string   `env:"USERNAME,required"`
	Password      string   `env:"PASSWORD,required"`
	Email         string   `env:"EMAIL,required"`
	WorkspaceName string   `env:"WORKSPACE_NAME" envDefault:"Main"`
}

type DatabaseConfig struct {
	Host         string `env:"HOST,required"`
	User         string `env:"USER,required"`
	Password     string `env:"PASSWORD,required"`
	Name         string `env:"NAME,required"`
	Port         int    `env:"PORT,required"`
	MaxOpenConns int    `env:"MAX_OPEN_CONN" envDefault:"5"`
	MaxIdleConns int    `env:"MAX_IDLE_CONN" envDefault:"2"`
}

type ServerConfig struct {
	Port           string `env:"PORT" envDefault:":4512"`
	URL            string `env:"URL"`
	AllowedOrigins string `env:"ALLOWED_ORIGINS"`
}

type Config struct {
	App      AppConfig      `envPrefix:"APP_"`
	Database DatabaseConfig `envPrefix:"DATABASE_"`
	Server   ServerConfig   `envPrefix:"SERVER_"`
}

func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ctx, log, err := slogx.NewWithContext(ctx)
	if err != nil {
		panic(err)
	}

	var cfg Config
	err = env.Parse(&cfg)
	if err != nil {
		log.Panic(err)
	}
	if cfg.Server.URL == "" {
		cfg.Server.URL = "http://0.0.0.0" + cfg.Server.Port
	}

	db, err := mariadb.Open(ctx, mariadb.Config{
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		User:         cfg.Database.User,
		Password:     cfg.Database.Password,
		DatabaseName: cfg.Database.Name,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Error("closing database connection", "error", err)
		}
	}()

	// system
	systemRepo := system.NewMariaDBRepository(db)
	systemService := system.NewService(systemRepo)

	// user
	userRepo := user.NewMariaDBRepository(db)
	userService := user.NewService(userRepo)

	// domain
	domainRepo := domain.NewMariaDBRepository(db)
	domainService := domain.NewService(domainRepo)

	// workspace
	workspaceRepo := workspace.NewMariaDBRepository(db)
	workspaceService := workspace.NewService(workspaceRepo, userService)

	// collection
	collectionRepo := collection.NewMariaDBRepository(db)
	collectionService := collection.NewService(collectionRepo)

	err = serverBootstrap(ctx, cfg, systemService, userService, domainService, workspaceService)
	if err != nil {
		log.Panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(middleware.Compress(5))
	r.Use(middleware.AllowContentType("application/json"))

	r.Route("/api", func(r chi.Router) {
		// Root handlers
		api.NewHandler(r)

		// API v1 handlers
		r.Route("/v1", func(r chi.Router) {
			apiv1.NewCollectionHandler(r, collectionService)
		})
	})

	errLog := slog.NewLogLogger(log.Handler(), slog.LevelError)
	server := http.Server{
		Handler:      r,
		Addr:         cfg.Server.Port,
		ErrorLog:     errLog,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("server listening and serving on " + cfg.Server.Port)

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Panic(err)
		}
	}()

	// Interrupt signal
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		log.Panic(err)
	}

	log.Info("server stopped")
}

// serverBootstrap initializes the server.
func serverBootstrap(
	ctx context.Context,
	cfg Config,
	systemService system.Service,
	userService user.Service,
	domainService domain.Service,
	wsService workspace.Service,
) error {
	log := slogx.FromCtx(ctx)
	log.Debug("bootstrapping server")

	var userEntity user.User
	systemEntity, err := systemService.Info(ctx)
	if err != nil {
		if !errors.Is(err, errs.ErrNotFound) {
			return fmt.Errorf("loading system info: %w", err)
		}
		log.Info("system configuration not found, initiating first-time setup")

		// Find or create the initial user
		var created bool
		userEntity, created, err = userService.FindOrCreateByUsername(ctx, user.User{
			FirstName:     cfg.App.Username,
			Username:      cfg.App.Username,
			Password:      cfg.App.Password, // TODO: hash the password
			Email:         cfg.App.Email,
			EmailVerified: true,
		})
		if err != nil {
			return fmt.Errorf("creating default user: %w", err)
		}
		if created {
			log.Info("system user created", "user", userEntity.Username)
		} else {
			log.Info("system user already exists, using existing account", "user", userEntity.Username)
		}

		// Set this user as the immutable owner of the system
		systemEntity, err = systemService.Configure(ctx, userEntity.ID)
		if err != nil {
			return fmt.Errorf("configuring system: %w", err)
		}
		log.Info("system configured")
	}

	if userEntity.ID != systemEntity.OwnerUserID {
		userEntity, err = userService.FindByID(ctx, systemEntity.OwnerUserID)
		if err != nil {
			return fmt.Errorf("loading system owner: %w", err)
		}
	}

	// After the initialization of a qorvin database, the owner/primary user cannot be changed.
	if userEntity.Username != cfg.App.Username {
		return errors.New("local configuration username does not match registered system owner")
	}

	for _, dom := range cfg.App.Domains {
		domainEntity, created, err := domainService.FindOrCreateByDomain(ctx, domain.Domain{
			Domain: dom,
		})
		if err != nil {
			return fmt.Errorf("configuring domain: %w", err)
		}
		if created {
			log.Info("system domain configured", "domain", dom)
		} else {
			log.Info("system domain already exists, using existing one", "domain", dom)
		}

		wsEntity, created, err := wsService.FindOrCreateByDomainID(ctx, workspace.Workspace{
			Name:        cfg.App.WorkspaceName,
			Slug:        strings.ToLower(cfg.App.Username) + "-workspace-" + domainEntity.ID.String(), // TODO: apply slugfy
			Description: "Default workspace for " + cfg.App.Username,
			DomainID:    domainEntity.ID,
		})
		if err != nil {
			return fmt.Errorf("configuring workspace: %w", err)
		}
		if created {
			log.Info("system workspace configured", "workspace", cfg.App.WorkspaceName)
		} else {
			log.Info("system workspace already exists, using existing one", "workspace", cfg.App.WorkspaceName)
		}

		err = wsService.AddUser(ctx, workspace.WorkspaceUser{
			WorkspaceID: wsEntity.ID,
			UserID:      userEntity.ID,
			Role:        workspace.AdminRole,
		})
		if err != nil {
			return fmt.Errorf("adding owner access to workspace: %w", err)
		}
		log.Info("system owner workspace access ensured", "workspace", wsEntity.Name, "user", userEntity.Username)
	}

	log.Info("server bootstrap complete")
	return nil
}
