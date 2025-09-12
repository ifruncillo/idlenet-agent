package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ifruncillo/idlenet-agent/internal/config"
	"github.com/ifruncillo/idlenet-agent/internal/heartbeat"
)

var (
	// Overridden at build time via -ldflags "-X 'main.version=vX.Y.Z'"
	version = "dev"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[idlenet] ")

	// Flags
	defaultAPI := os.Getenv("IDLENET_API_BASE")
	if defaultAPI == "" {
		defaultAPI = "http://127.0.0.1:8787" // convenient for local stub
	}
	apiBase := flag.String("api-base", defaultAPI, "Base URL for API (e.g., https://yourdomain)")
	email := flag.String("email", "", "Account email (required first run)")
	referral := flag.String("referral", "", "Optional referral code on first run")
	interval := flag.Duration("interval", 60*time.Second, "Heartbeat interval")
	showVer := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVer {
		fmt.Printf("IdleNet Agent %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	// Load or create config (generates deviceID on first run).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	cfgChanged := false

	// First run: capture email/referral if provided.
	if cfg.Email == "" && *email != "" {
		cfg.Email = *email
		cfgChanged = true
	}
	if cfg.Referral == "" && *referral != "" {
		cfg.Referral = *referral
		cfgChanged = true
	}
	if cfgChanged {
		if err := config.Save(cfg); err != nil {
			log.Fatalf("save config: %v", err)
		}
	}

	// Enforce email presence for API calls.
	if cfg.Email == "" {
		log.Fatalf("email is required on first run: pass --email you@example.com")
	}

	confPath, _ := config.FilePath()
	log.Printf("config: %s", confPath)
	log.Printf("deviceId: %s", cfg.DeviceID)

	// Client
	hb := heartbeat.NewClient(*apiBase, version)

	// Register once (idempotent on your backend).
	if !cfg.Registered {
		if err := hb.Register(context.Background(), cfg.Email, cfg.DeviceID, cfg.Referral); err != nil {
			log.Fatalf("register failed: %v", err)
		}
		cfg.Registered = true
		_ = config.Save(cfg)
		log.Printf("registered OK")
	} else {
		// Optional best-effort "reconfirm" in case backend was wiped; ignore errors.
		_ = hb.Register(context.Background(), cfg.Email, cfg.DeviceID, cfg.Referral)
	}

	// Heartbeat loop with graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Send one immediately.
	go func() {
		if err := hb.Beat(ctx, cfg.Email, cfg.DeviceID); err != nil {
			log.Printf("heartbeat error: %v", err)
		} else {
			log.Printf("heartbeat OK")
		}
	}()

	tick := time.NewTicker(*interval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("shutting down gracefully")
			return
		case <-tick.C:
			if err := hb.Beat(ctx, cfg.Email, cfg.DeviceID); err != nil {
				log.Printf("heartbeat error: %v", err)
			} else {
				log.Printf("heartbeat OK")
			}
		}
	}
}

