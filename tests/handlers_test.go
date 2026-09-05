package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"kumite/db"
	"kumite/engine"
	"kumite/handlers"
	"kumite/models"
)

// githubFixtureServer stands in for raw.githubusercontent.com with ETag
// support for conditional requests.
func githubFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	etag := ""
	mux := http.NewServeMux()
	mux.HandleFunc("/tree", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tree":[
			{"path":"engineering/engineering-software-architect.md","type":"blob"},
			{"path":"design/design-ux-researcher.md","type":"blob"},
			{"path":"README.md","type":"blob"}
		]}`)
	})
	mux.HandleFunc("/arch.md", func(w http.ResponseWriter, r *http.Request) {
		if etag != "" && r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		etag = `"v1"`
		w.Header().Set("ETag", etag)
		fmt.Fprint(w, "# Architect prompt v1")
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func newHandlerServer(t *testing.T, ttl time.Duration) (*db.DB, *httptest.Server) {
	t.Helper()
	d, err := db.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mock := NewMockLLM()
	t.Cleanup(mock.Server.Close)

	// The roster is loaded from the repo file — same content main.go embeds.
	rosterRaw, err := os.ReadFile("../roster/agents.json")
	if err != nil {
		t.Fatalf("read roster: %v", err)
	}
	var roster []models.RosterEntry
	if err := json.Unmarshal(rosterRaw, &roster); err != nil {
		t.Fatalf("parse roster: %v", err)
	}

	srv := &handlers.Server{
		DB:          d,
		LLM:         engine.NewClient(),
		HTTP:        &http.Client{Timeout: 10 * time.Second},
		CacheTTL:    ttl,
		Roster:      roster,
		LLMOverride: &engine.RuntimeLLMConfig{Endpoint: mock.Server.URL, Model: "mock-model"},
	}
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return d, ts
}

func doJSON(t *testing.T, method, url string, body any) (*http.Response, map[string]any, []byte) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read response: %v", readErr)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	// Keep the response readable for the caller's assertions.
	r := resp
	return r, m, raw
}

func TestSessionHTTPCRUD(t *testing.T) {
	_, ts := newHandlerServer(t, time.Hour)

	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "Field Ledger",
		"raw_input":    "raw material",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d", resp.StatusCode)
	}
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatal("no id returned")
	}

	resp, _, raw := doJSON(t, "GET", ts.URL+"/api/sessions/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d", resp.StatusCode)
	}
	var sess models.Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if sess.Phase != models.PhaseIntake || sess.RawSource != "" || sess.RawInput != "raw material" {
		t.Fatalf("session = %+v", sess)
	}

	resp, m, _ = doJSON(t, "PATCH", ts.URL+"/api/sessions/"+id, map[string]any{
		"project_name": "Renamed",
		"tags":         []string{"field"},
		"domain":       "tool",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", resp.StatusCode)
	}
	resp, _, raw = doJSON(t, "GET", ts.URL+"/api/sessions/"+id, nil)
	_ = json.Unmarshal(raw, &sess)
	if sess.ProjectName != "Renamed" || sess.Domain != "tool" || len(sess.Tags) != 1 {
		t.Fatalf("patched session = %+v", sess)
	}

	resp, _, _ = doJSON(t, "DELETE", ts.URL+"/api/sessions/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	resp, _, _ = doJSON(t, "GET", ts.URL+"/api/sessions/"+id, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want 404", resp.StatusCode)
	}
}

func TestSessionsListFilterHTTP(t *testing.T) {
	_, ts := newHandlerServer(t, time.Hour)

	doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{"project_name": "A", "raw_input": "x"})
	_, mB, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{"project_name": "B", "raw_input": "x"})
	idB := mB["id"].(string)
	doJSON(t, "PATCH", ts.URL+"/api/sessions/"+idB, map[string]any{"domain": "game"})

	resp, _, raw := doJSON(t, "GET", ts.URL+"/api/sessions?domain=game", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", resp.StatusCode)
	}
	var items []models.SessionListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(items) != 1 || items[0].ProjectName != "B" {
		t.Fatalf("filtered list = %+v", items)
	}
}

func TestConfigHTTPMaskAndPut(t *testing.T) {
	d, ts := newHandlerServer(t, time.Hour)

	resp, cfg, _ := doJSON(t, "GET", ts.URL+"/api/config", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get config status = %d", resp.StatusCode)
	}
	if cfg["llm_api_key"] != "" {
		t.Fatalf("empty key should mask to empty, got %v", cfg["llm_api_key"])
	}

	doJSON(t, "PUT", ts.URL+"/api/config", map[string]any{
		"llm_model":   "llama3:70b",
		"llm_api_key": "sk-real-key",
	})
	resp, cfg, _ = doJSON(t, "GET", ts.URL+"/api/config", nil)
	if cfg["llm_model"] != "llama3:70b" {
		t.Fatalf("model not updated: %v", cfg["llm_model"])
	}
	if cfg["llm_api_key"] != "••••••••" {
		t.Fatalf("key must be masked, got %v", cfg["llm_api_key"])
	}

	// A client round-tripping the masked value must not clobber the real key.
	doJSON(t, "PUT", ts.URL+"/api/config", map[string]any{"llm_api_key": "••••••••"})
	stored, _ := d.AllConfig()
	if stored["llm_api_key"] != "sk-real-key" {
		t.Fatalf("masked value overwrote the real key: %q", stored["llm_api_key"])
	}
}

func TestRosterHTTP(t *testing.T) {
	_, ts := newHandlerServer(t, time.Hour)

	resp, _, raw := doJSON(t, "GET", ts.URL+"/api/agents/roster", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("roster status = %d", resp.StatusCode)
	}
	var entries []models.RosterEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("decode roster: %v", err)
	}
	if len(entries) != 13 {
		t.Fatalf("roster entries = %d, want 13", len(entries))
	}
	for _, e := range entries {
		if e.AgentID == "testing-reality-checker" && !e.Fixed {
			t.Fatal("reality checker must be fixed")
		}
	}
}

func TestAgentPromptFetchAndCache(t *testing.T) {
	gh := githubFixtureServer(t)
	_, ts := newHandlerServer(t, time.Hour)
	url := ts.URL + "/api/agents/arch?source_url=" + gh.URL + "/arch.md"

	resp, m, _ := doJSON(t, "GET", url, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch status = %d", resp.StatusCode)
	}
	if m["cached"] != false || m["stale"] != false || m["content"] != "# Architect prompt v1" {
		t.Fatalf("first fetch = %v", m)
	}

	resp, m, _ = doJSON(t, "GET", url, nil)
	if m["cached"] != true || m["stale"] != false {
		t.Fatalf("second fetch should hit fresh cache: %v", m)
	}
}

// TestAgentPromptConditional304: TTL expired + unchanged upstream answers 304,
// which resets the TTL and serves the cache as fresh.
func TestAgentPromptConditional304(t *testing.T) {
	gh := githubFixtureServer(t)
	_, ts := newHandlerServer(t, 0) // TTL 0 → every request revalidates
	url := ts.URL + "/api/agents/arch?source_url=" + gh.URL + "/arch.md"

	doJSON(t, "GET", url, nil) // populates cache with ETag
	resp, m, _ := doJSON(t, "GET", url, nil)
	if resp.StatusCode != http.StatusOK || m["cached"] != true || m["stale"] != false {
		t.Fatalf("304 path = %d %v", resp.StatusCode, m)
	}
}

// TestAgentPromptStaleOnUnreachable: expired cache + dead upstream serves the
// stale content with stale=true rather than failing.
func TestAgentPromptStaleOnUnreachable(t *testing.T) {
	gh := githubFixtureServer(t)
	_, ts := newHandlerServer(t, 0)
	url := ts.URL + "/api/agents/arch?source_url=" + gh.URL + "/arch.md"

	doJSON(t, "GET", url, nil)
	gh.Close() // upstream dies after a successful fetch

	resp, m, _ := doJSON(t, "GET", url, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stale fetch status = %d", resp.StatusCode)
	}
	if m["stale"] != true || m["cached"] != true || m["content"] != "# Architect prompt v1" {
		t.Fatalf("stale fetch = %v", m)
	}
}

func TestCustomAgentsHTTP(t *testing.T) {
	_, ts := newHandlerServer(t, time.Hour)

	body := map[string]any{
		"display_name":    "Domain Expert",
		"role_summary":    "Analyzes domain X.",
		"wave_preference": 2,
		"system_prompt":   "You are a domain expert.",
	}
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/agents/custom", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d (%v)", resp.StatusCode, m)
	}
	id := m["id"].(string)
	if id != "domain-expert" {
		t.Fatalf("slug id = %q", id)
	}

	// Slug collision gets a distinct id.
	resp, m2, _ := doJSON(t, "POST", ts.URL+"/api/agents/custom", body)
	if resp.StatusCode != http.StatusOK || m2["id"] == id {
		t.Fatalf("duplicate slug not deduplicated: %v", m2)
	}

	resp, _, raw := doJSON(t, "GET", ts.URL+"/api/agents/custom", nil)
	var agents []models.CustomAgent
	_ = json.Unmarshal(raw, &agents)
	if len(agents) != 2 {
		t.Fatalf("custom agents = %d", len(agents))
	}

	// The custom agent's prompt is served locally, no GitHub fetch.
	resp, m3, _ := doJSON(t, "GET", ts.URL+"/api/agents/"+id, nil)
	if resp.StatusCode != http.StatusOK || m3["content"] != "You are a domain expert." {
		t.Fatalf("custom prompt fetch = %d %v", resp.StatusCode, m3)
	}

	doJSON(t, "PUT", ts.URL+"/api/agents/custom/"+id, map[string]any{
		"display_name": "Domain Expert", "role_summary": "Updated.",
		"wave_preference": 1, "system_prompt": "You are a domain expert v2.",
	})
	resp, m4, _ := doJSON(t, "GET", ts.URL+"/api/agents/"+id, nil)
	if resp.StatusCode != http.StatusOK || m4["content"] != "You are a domain expert v2." {
		t.Fatalf("after update = %d %v", resp.StatusCode, m4)
	}

	resp, _, _ = doJSON(t, "DELETE", ts.URL+"/api/agents/custom/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
	// API.md defines no GET /api/agents/custom/{id} — the agent-prompt route
	// resolves custom agents and must 404 after deletion.
	resp, _, _ = doJSON(t, "GET", ts.URL+"/api/agents/"+id, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted custom agent still resolvable (%d)", resp.StatusCode)
	}

	// Missing required fields → 400.
	resp, _, _ = doJSON(t, "POST", ts.URL+"/api/agents/custom", map[string]any{"display_name": "x"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d", resp.StatusCode)
	}
}

func TestHealthHTTP(t *testing.T) {
	_, ts := newHandlerServer(t, time.Hour)
	resp, m, _ := doJSON(t, "GET", ts.URL+"/health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}
	if m["ok"] != true || m["model"] != "mock-model" {
		t.Fatalf("health = %v", m)
	}
}
