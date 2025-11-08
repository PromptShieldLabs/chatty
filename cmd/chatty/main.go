package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/PromptShieldLabs/chatty/internal"
	"github.com/PromptShieldLabs/chatty/internal/config"
	"github.com/PromptShieldLabs/chatty/internal/storage"
)

var (
	version = "0.1.5"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	client, err := internal.NewClient(cfg.API.Key, cfg.API.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create client: %v\n", err)
		os.Exit(1)
	}

	// Initialize persistence store (optional)
	var store *storage.Store
	if cfg.Storage.Path != "disable" {
		store, err = storage.Open(cfg.Storage.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: persistence disabled (%v)\n", err)
		} else {
			defer store.Close()
		}
	}

	// Create chat session with version info
	versionInfo := version
	if commit != "none" && commit != "" {
		versionInfo = fmt.Sprintf("%s (build %s)", version, commit)
	}
	session, err := internal.NewSession(client, cfg, store, versionInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create session: %v\n", err)
		os.Exit(1)
	}

	// Run the chat loop
	ctx := context.Background()
	if err := session.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
