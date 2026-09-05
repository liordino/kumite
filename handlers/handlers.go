package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"kumite/db"
	"kumite/engine"
	"kumite/models"
)

// Server carries the handler dependencies. No global state — everything is
// passed explicitly and the same struct serves the real binary and tests.
type Server struct {
	DB       *db.DB
	LLM      *engine.Client
	HTTP     *http.Client // used for GitHub fetches
	CacheTTL time.Duration
	Roster   []models.RosterEntry // curated roster, embedded at build time

	// hub tracks in-flight runs and their SSE subscribers; lazily created.
	hub *runHub

	// LLMOverride, when set (LLM_MOCK=true), replaces the app_config-derived
	// provider settings so dev runs never touch a real provider.
	LLMOverride *engine.RuntimeLLMConfig
}

// Routes builds the Chi router. Intake and pipeline route groups are added
// with their milestones; unimplemented paths 404.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", s.handleHealth)

	r.Route("/api/sessions", func(r chi.Router) {
		r.Get("/", s.listSessions)
		r.Post("/", s.createSession)
		r.Get("/{id}", s.getSession)
		r.Patch("/{id}", s.patchSession)
		r.Delete("/{id}", s.deleteSession)
	})

	r.Route("/api/agents", func(r chi.Router) {
		r.Get("/roster", s.roster)
		r.Route("/custom", func(r chi.Router) {
			r.Get("/", s.listCustomAgents)
			r.Post("/", s.createCustomAgent)
			r.Put("/{id}", s.updateCustomAgent)
			r.Delete("/{id}", s.deleteCustomAgent)
		})
		r.Get("/{agentId}", s.getAgentPrompt)
	})

	r.Route("/api/intake", func(r chi.Router) {
		r.Get("/explain", s.intakeExplain)
		r.Post("/{id}", s.runIntake)
		r.Delete("/{id}", s.revertIntake)
	})

	r.Route("/api/pipeline", func(r chi.Router) {
		r.Post("/phase0/{id}", s.phase0)
		r.Patch("/{id}/plan", s.updatePlan)
		r.Post("/run/{id}", s.run)
		r.Post("/resume/{id}", s.resume)
		r.Get("/stream/{id}", s.stream)
	})

	r.Route("/api/config", func(r chi.Router) {
		r.Get("/", s.getConfig)
		r.Put("/", s.putConfig)
	})

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	cfg, err := s.llmConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"model":    cfg.Model,
		"endpoint": cfg.Endpoint,
	})
}

// llmConfig resolves the provider settings: the mock override when active,
// otherwise the runtime values from app_config (switchable without restart).
func (s *Server) llmConfig() (engine.RuntimeLLMConfig, error) {
	if s.LLMOverride != nil {
		return *s.LLMOverride, nil
	}
	rt, err := s.DB.LoadRuntimeLLM()
	if err != nil {
		return engine.RuntimeLLMConfig{}, err
	}
	return engine.RuntimeLLMConfig{Endpoint: rt.Endpoint, Model: rt.Model, APIKey: rt.APIKey}, nil
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}
