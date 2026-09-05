package handlers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"kumite/db"
	"kumite/engine"
	"kumite/models"
)

// rosterPath is dropped — the roster is embedded at build time and handed to
// the server explicitly (see Server.Roster).

// roster implements GET /api/agents/roster — the full built-in roster.
func (s *Server) roster(w http.ResponseWriter, _ *http.Request) {
	entries := s.Roster
	if entries == nil {
		entries = []models.RosterEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// getAgentPrompt implements GET /api/agents/:agentId. Resolution order:
// query source_url override, then the built-in roster, then a custom agent's
// locally stored system prompt (no GitHub fetch for custom agents).
func (s *Server) getAgentPrompt(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	overrideURL := r.URL.Query().Get("source_url")

	if overrideURL != "" {
		s.servePrompt(w, r, agentID, overrideURL)
		return
	}

	if entry, ok := s.rosterEntry(agentID); ok {
		s.servePrompt(w, r, agentID, entry.SourceURL)
		return
	}

	custom, err := s.DB.GetCustomAgent(agentID)
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "agent not found in roster or custom agents")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": custom.ID,
		"content":  custom.SystemPrompt,
		"cached":   false,
		"stale":    false,
	})
}

func (s *Server) servePrompt(w http.ResponseWriter, r *http.Request, agentID, sourceURL string) {
	content, cached, stale, err := engine.FetchAgentPrompt(r.Context(), s.DB, s.HTTP, agentID, sourceURL, s.CacheTTL)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": agentID,
		"content":  content,
		"cached":   cached,
		"stale":    stale,
	})
}

// rosterEntry finds one roster entry by agent_id.
func (s *Server) rosterEntry(agentID string) (models.RosterEntry, bool) {
	for _, e := range s.Roster {
		if e.AgentID == agentID {
			return e, true
		}
	}
	return models.RosterEntry{}, false
}

// --- Custom agents ---

func (s *Server) listCustomAgents(w http.ResponseWriter, _ *http.Request) {
	agents, err := s.DB.ListCustomAgents()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agents == nil {
		agents = []models.CustomAgent{}
	}
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) createCustomAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName    string `json:"display_name"`
		RoleSummary    string `json:"role_summary"`
		WavePreference int    `json:"wave_preference"`
		SystemPrompt   string `json:"system_prompt"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(body.DisplayName) == "" || strings.TrimSpace(body.SystemPrompt) == "" {
		writeErr(w, http.StatusBadRequest, "display_name and system_prompt are required")
		return
	}
	if body.WavePreference != 1 && body.WavePreference != 2 {
		body.WavePreference = 1
	}

	agent := models.CustomAgent{
		ID:             slugify(body.DisplayName),
		DisplayName:    body.DisplayName,
		RoleSummary:    body.RoleSummary,
		WavePreference: body.WavePreference,
		SystemPrompt:   body.SystemPrompt,
	}
	// Slug collision: same display name twice gets distinct ids.
	if _, err := s.DB.GetCustomAgent(agent.ID); err == nil {
		agent.ID = agent.ID + "-" + db.NewID()[:8]
	}
	created, err := s.DB.CreateCustomAgent(agent)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": created.ID})
}

func (s *Server) updateCustomAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		DisplayName    string `json:"display_name"`
		RoleSummary    string `json:"role_summary"`
		WavePreference int    `json:"wave_preference"`
		SystemPrompt   string `json:"system_prompt"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if body.WavePreference != 1 && body.WavePreference != 2 {
		body.WavePreference = 1
	}
	agent := models.CustomAgent{
		ID:             id,
		DisplayName:    body.DisplayName,
		RoleSummary:    body.RoleSummary,
		WavePreference: body.WavePreference,
		SystemPrompt:   body.SystemPrompt,
	}
	err := s.DB.UpdateCustomAgent(agent)
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "custom agent not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) deleteCustomAgent(w http.ResponseWriter, r *http.Request) {
	err := s.DB.DeleteCustomAgent(chi.URLParam(r, "id"))
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "custom agent not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// slugify turns a display name into a url-safe id ("Domain Expert" ->
// "domain-expert").
func slugify(name string) string {
	lower := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = db.NewID()
	}
	return slug
}
