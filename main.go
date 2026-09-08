// Kumite — self-hosted multi-agent feasibility consultancy.
//
// main.go wires the Chi router, opens the database, runs migrations and
// startup reconciliation, and serves the API. LLM_MOCK=true swaps the
// provider for the embedded mock so the full stack runs without inference.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof on the diagnostic listener
	"runtime"
	"time"

	"kumite/config"
	"kumite/db"
	"kumite/engine"
	"kumite/handlers"
	"kumite/models"
	"kumite/tests"
)

// The curated 13-agent roster is a repo artifact (AGENTS.md) and is embedded
// into the binary at build time, so the served roster never depends on the
// process working directory.
//
//go:embed roster/agents.json
var rosterJSON []byte

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := database.SeedLLMDefaults(cfg.LLMEndpoint, cfg.LLMModel, cfg.LLMAPIKey); err != nil {
		log.Fatalf("seed config: %v", err)
	}
	// Single process — sessions in running/synthesis at boot are orphaned by
	// definition. Move them to interrupted, reset running nodes to pending,
	// keep every partial output.
	if err := database.ReconcileOrphanedRuns(); err != nil {
		log.Fatalf("reconcile: %v", err)
	}

	var roster []models.RosterEntry
	if err := json.Unmarshal(rosterJSON, &roster); err != nil {
		log.Fatalf("parse roster: %v", err)
	}

	var repoTreeURL string
	if treeURL, ok := engine.RepoTreeURL(cfg.GitHubRawBase); ok {
		repoTreeURL = treeURL
	}

	var override *engine.RuntimeLLMConfig
	if cfg.LLMMock {
		mock := tests.NewMockLLM()
		override = &engine.RuntimeLLMConfig{
			Endpoint: mock.Server.URL,
			Model:    "mock-model",
		}
		log.Printf("LLM_MOCK=true — using embedded mock provider at %s", mock.Server.URL)
	}

	server := &handlers.Server{
		DB:            database,
		LLM:           engine.NewClient(),
		HTTP:          &http.Client{Timeout: 30 * time.Second},
		CacheTTL:      time.Duration(cfg.AgentCacheTTLHours) * time.Hour,
		GitHubRawBase: cfg.GitHubRawBase,
		GitHubTreeURL: repoTreeURL,
		Roster:        roster,
		LLMOverride:   override,
	}

	// Diagnostic-only listener (NOT part of the API surface): pprof goroutine
	// and CPU profiles on 127.0.0.1, for post-morteming wedges like the one
	// that would otherwise be undebuggable. Never exposed beyond localhost.
	diag := http.NewServeMux()
	diag.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	go func() {
		log.Printf("diagnostics on http://127.0.0.1:3002/debug/pprof/")
		if err := http.ListenAndServe("127.0.0.1:3002", diag); err != nil {
			log.Printf("diagnostics listener: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("kumite listening on %s (db: %s, goroutines: %d)", addr, cfg.DBPath, runtime.NumGoroutine())
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
