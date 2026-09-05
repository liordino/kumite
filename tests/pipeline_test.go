package tests

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"kumite/db"
	"kumite/engine"
	"kumite/handlers"
	"kumite/models"
)

// newPipelineServer returns the db handle, the mock provider (for route
// registration) and the HTTP server under test.
func newPipelineServer(t *testing.T, ttl time.Duration, configure ...func(*handlers.Server)) (*db.DB, *MockLLM, *httptest.Server) {
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
	for _, c := range configure {
		c(srv)
	}
	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return d, mock, ts
}

func sseEvent(t *testing.T, name string, events []sseLine) map[string]any {
	t.Helper()
	for i := range events {
		if events[i].Event == name {
			var m map[string]any
			if err := json.Unmarshal([]byte(events[i].Data), &m); err != nil {
				t.Fatalf("decode %s data: %v", name, err)
			}
			return m
		}
	}
	return nil
}

type sseLine struct {
	Event string
	Data  string
}

// readSSE POSTs and consumes the stream until the server closes it.
func readSSE(t *testing.T, url string) []sseLine {
	t.Helper()
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d", resp.StatusCode)
	}
	var out []sseLine
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var cur sseLine
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			cur = sseLine{Event: strings.TrimPrefix(line, "event: ")}
		case strings.HasPrefix(line, "data: "):
			cur.Data = strings.TrimPrefix(line, "data: ")
			out = append(out, cur)
			cur = sseLine{}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read stream: %v", err)
	}
	return out
}

// registerPipelineRoutes wires the mock: phase 0 first (its prompt contains
// every roster id), then phase 2 (its prompt contains every agent id via the
// outputs JSON), then per-agent routes (each prompt carries exactly its own
// agent_id in the task envelope).
func registerPipelineRoutes(mock *MockLLM, phase0Fixture, agentFixture, psdFixture string) {
	mock.Register(ContentContains("INVOCATION: Phase 0"), func(MockRequest) string {
		return phase0Fixture
	})
	mock.Register(ContentContains("INVOCATION: Phase 2"), func(MockRequest) string {
		return psdFixture
	})
	for _, id := range []string{
		"engineering-software-architect",
		"design-ux-researcher",
		"project-manager-senior",
		"testing-reality-checker",
	} {
		needle := `"agent_id": "` + id + `"`
		mock.Register(ContentContains(needle), func(MockRequest) string {
			return agentFixture
		})
	}
}

func createSessionForRun(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "Field Ledger",
		"raw_input":    "A small tool for logging field observations offline, with CSV export.",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create session status = %d", resp.StatusCode)
	}
	return m["id"].(string)
}

func runPhase0(t *testing.T, ts *httptest.Server, id string) *models.PipelinePlan {
	t.Helper()
	resp, m, raw := doJSON(t, "POST", ts.URL+"/api/pipeline/phase0/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("phase0 status = %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		PipelinePlan *models.PipelinePlan `json:"pipeline_plan"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode phase0: %v", err)
	}
	if out.PipelinePlan == nil {
		t.Fatalf("no plan in response: %v", m)
	}
	return out.PipelinePlan
}

func TestPhase0HTTP(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	registerPipelineRoutes(mock, loadFixture(t, "phase0_plan.json"), loadFixture(t, "agent_output.json"), loadFixture(t, "psd.md"))

	id := createSessionForRun(t, ts)
	plan := runPhase0(t, ts, id)

	if plan.ProjectName != "Field Ledger" || plan.Context.Domain != "tool" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Pipeline.Wave1) != 2 || len(plan.Pipeline.Wave2) != 1 || len(plan.Pipeline.Fixed) != 1 {
		t.Fatalf("waves = %d/%d/%d, want 2/1/1",
			len(plan.Pipeline.Wave1), len(plan.Pipeline.Wave2), len(plan.Pipeline.Fixed))
	}
	rc := plan.Pipeline.Fixed[0]
	if rc.AgentID != "testing-reality-checker" || !rc.Enabled || rc.Wave != models.Fixed {
		t.Fatalf("reality checker node = %+v", rc)
	}

	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.Phase != models.PhasePipelineReview {
		t.Fatalf("phase = %q", sess.Phase)
	}
	if sess.Domain != "tool" || len(sess.Tags) != 3 {
		t.Fatalf("domain/tags columns not synced: %q %+v", sess.Domain, sess.Tags)
	}
}

func TestPhase0Validation(t *testing.T) {
	_, mock, ts := newPipelineServer(t, time.Hour)
	registerPipelineRoutes(mock, loadFixture(t, "phase0_plan.json"), loadFixture(t, "agent_output.json"), loadFixture(t, "psd.md"))

	if resp, _, _ := doJSON(t, "POST", ts.URL+"/api/pipeline/phase0/no-such-session", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown session status = %d", resp.StatusCode)
	}

	// Empty raw_input → 422.
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{"project_name": "Empty", "raw_input": ""})
	id := m["id"].(string)
	resp, _, _ = doJSON(t, "POST", ts.URL+"/api/pipeline/phase0/"+id, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("empty input status = %d, want 422", resp.StatusCode)
	}
}

func TestUpdatePlanValidation(t *testing.T) {
	_, mock, ts := newPipelineServer(t, time.Hour)
	registerPipelineRoutes(mock, loadFixture(t, "phase0_plan.json"), loadFixture(t, "agent_output.json"), loadFixture(t, "psd.md"))

	id := createSessionForRun(t, ts)
	plan := runPhase0(t, ts, id)

	// Valid modification: drop the UX researcher from Wave 1.
	modified := *plan
	modified.Pipeline.Wave1 = plan.Pipeline.Wave1[:1]
	resp, _, _ := doJSON(t, "PATCH", ts.URL+"/api/pipeline/"+id+"/plan", modified)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid patch status = %d", resp.StatusCode)
	}

	// Unknown agent id → 422.
	bad := *plan
	bad.Pipeline.Wave2 = []models.AgentNode{nodeFixture("no-such-agent", models.Wave2)}
	if resp, _, _ = doJSON(t, "PATCH", ts.URL+"/api/pipeline/"+id+"/plan", bad); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unresolvable id status = %d, want 422", resp.StatusCode)
	}

	// Empty Fixed wave → 409.
	emptyFixed := *plan
	emptyFixed.Pipeline.Fixed = nil
	if resp, _, _ = doJSON(t, "PATCH", ts.URL+"/api/pipeline/"+id+"/plan", emptyFixed); resp.StatusCode != http.StatusConflict {
		t.Fatalf("empty fixed wave status = %d, want 409", resp.StatusCode)
	}

	// Disabled Reality Checker → 409.
	rcOff := *plan
	rcOff.Pipeline.Fixed[0].Enabled = false
	if resp, _, _ = doJSON(t, "PATCH", ts.URL+"/api/pipeline/"+id+"/plan", rcOff); resp.StatusCode != http.StatusConflict {
		t.Fatalf("disabled reality checker status = %d, want 409", resp.StatusCode)
	}

	// Reality Checker moved to Wave 1 (absent from Fixed) → 409.
	moved := *plan
	rcNode := plan.Pipeline.Fixed[0]
	moved.Pipeline.Fixed = nil
	rcNode.Wave = models.Wave1
	moved.Pipeline.Wave1 = append(append([]models.AgentNode(nil), plan.Pipeline.Wave1...), rcNode)
	if resp, _, _ = doJSON(t, "PATCH", ts.URL+"/api/pipeline/"+id+"/plan", moved); resp.StatusCode != http.StatusConflict {
		t.Fatalf("moved reality checker status = %d, want 409", resp.StatusCode)
	}
}

func TestRunSSE(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	registerPipelineRoutes(mock, loadFixture(t, "phase0_plan.json"), loadFixture(t, "agent_output.json"), loadFixture(t, "psd.md"))

	id := createSessionForRun(t, ts)
	runPhase0(t, ts, id)

	events := readSSE(t, ts.URL+"/api/pipeline/run/"+id)
	for _, e := range events {
		if e.Event == "agent_error" || e.Event == "pipeline_error" {
			t.Logf("DEBUG %s: %s", e.Event, e.Data)
		}
	}
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = e.Event
	}

	expect := []string{
		"pipeline_start",
		"agent_start", "agent_done", // arch (wave 1)
		"agent_start", "agent_done", // ux (wave 1)
		"wave_complete",             // wave 1
		"agent_start", "agent_done", // pm (wave 2)
		"wave_complete",             // wave 2
		"agent_start", "agent_done", // reality checker (fixed)
		"wave_complete", // fixed
		"synthesis_start", "psd_chunk", "psd_done", "pipeline_complete",
	}
	pos := 0
	for _, want := range expect {
		found := false
		for ; pos < len(names); pos++ {
			if names[pos] == want {
				found = true
				pos++
				break
			}
		}
		if !found {
			t.Fatalf("event %q missing or out of order\ngot: %v", want, names)
		}
	}

	startData := sseEvent(t, "pipeline_start", events)
	if startData != nil && int(startData["total_agents"].(float64)) != 4 {
		t.Fatalf("total_agents = %v, want 4", startData["total_agents"])
	}

	// Durability: everything landed in the database.
	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.Phase != models.PhaseComplete {
		t.Fatalf("phase = %q, want complete", sess.Phase)
	}
	if len(sess.AgentOutputs) != 4 {
		t.Fatalf("agent outputs = %d, want 4", len(sess.AgentOutputs))
	}
	if sess.Psd == "" || !strings.Contains(sess.Psd, "# Project Summary Document: Field Ledger") {
		t.Fatalf("psd not persisted: %.80s", sess.Psd)
	}
	if sess.FindingSummary.Critical != 4 || sess.FindingSummary.Low != 4 {
		t.Fatalf("finding summary = %+v", sess.FindingSummary)
	}
	for _, node := range sess.PipelinePlan.AllNodes() {
		if node.Status != models.StatusDone {
			t.Fatalf("node %s status = %q, want done", node.ID, node.Status)
		}
	}

	// A completed session cannot run again.
	if resp, _, _ := doJSON(t, "POST", ts.URL+"/api/pipeline/run/"+id, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("second run status = %d, want 409", resp.StatusCode)
	}
}

func TestRunWrongPhase(t *testing.T) {
	_, _, ts := newPipelineServer(t, time.Hour)
	id := createSessionForRun(t, ts)
	if resp, _, _ := doJSON(t, "POST", ts.URL+"/api/pipeline/run/"+id, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("run on intake session status = %d, want 409", resp.StatusCode)
	}
}

// TestResumeHTTP: a run cut mid-flight resumes from node status — finished
// specialists are skipped (and reported in run_resumed), the rest execute,
// synthesis runs, and the session completes.
func TestResumeHTTP(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	registerPipelineRoutes(mock, loadFixture(t, "phase0_plan.json"), loadFixture(t, "agent_output.json"), loadFixture(t, "psd.md"))

	id := createSessionForRun(t, ts)
	plan := runPhase0(t, ts, id)

	// Simulate an interrupted run: both Wave 1 agents finished (outputs
	// persisted), the Wave 2 node was left running.
	for _, node := range plan.Pipeline.Wave1 {
		if err := d.AppendAgentOutput(id, agentOutputFixture(node.AgentID), node.ID, models.StatusDone); err != nil {
			t.Fatalf("persist partial output: %v", err)
		}
	}
	if err := d.SetNodeStatus(id, "project-manager-senior", models.StatusRunning); err != nil {
		t.Fatalf("set node running: %v", err)
	}
	if err := d.SetSessionPhase(id, models.PhaseInterrupted); err != nil {
		t.Fatalf("set interrupted: %v", err)
	}

	events := readSSE(t, ts.URL+"/api/pipeline/resume/"+id)

	resumed := sseEvent(t, "run_resumed", events)
	if resumed == nil {
		t.Fatalf("run_resumed missing; events: %v", events)
	}
	skipping, _ := resumed["skipping"].([]any)
	if len(skipping) != 2 {
		t.Fatalf("skipping = %v, want the two finished Wave 1 agents", resumed["skipping"])
	}
	if int(resumed["remaining"].(float64)) != 2 {
		t.Fatalf("remaining = %v, want 2", resumed["remaining"])
	}

	names := make([]string, len(events))
	for i, e := range events {
		names[i] = e.Event
	}
	for _, want := range []string{"agent_start", "agent_done", "synthesis_start", "psd_done", "pipeline_complete"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("event %q missing after resume\ngot: %v", want, names)
		}
	}

	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.Phase != models.PhaseComplete {
		t.Fatalf("phase = %q, want complete", sess.Phase)
	}
	if len(sess.AgentOutputs) != 4 {
		t.Fatalf("outputs = %d, want 4 (2 partial + 2 resumed)", len(sess.AgentOutputs))
	}
	// No duplicated outputs for the skipped agents.
	seen := map[string]int{}
	for _, o := range sess.AgentOutputs {
		seen[o.AgentID]++
	}
	for agentID, n := range seen {
		if n != 1 {
			t.Fatalf("agent %s produced %d outputs, want exactly 1", agentID, n)
		}
	}
}

// TestStreamAttachNoRun: attaching when nothing is in flight is a 409 —
// events are never replayed; the persisted session is the record.
func TestStreamAttachNoRun(t *testing.T) {
	_, _, ts := newPipelineServer(t, time.Hour)
	id := createSessionForRun(t, ts)
	if resp, _, _ := doJSON(t, "GET", ts.URL+"/api/pipeline/stream/"+id, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("attach with no run status = %d, want 409", resp.StatusCode)
	}
	if resp, _, _ := doJSON(t, "GET", ts.URL+"/api/pipeline/stream/no-such-session", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("attach unknown session status = %d, want 404", resp.StatusCode)
	}
}

// TestRepoListingHTTP: the advanced full-repo roster lists the repo's agent
// prompt files (README excluded), cached under the TTL.
func TestRepoListingHTTP(t *testing.T) {
	gh := githubFixtureServer(t)
	_, _, ts := newPipelineServer(t, time.Hour, func(s *handlers.Server) {
		s.GitHubRawBase = gh.URL
		s.GitHubTreeURL = gh.URL + "/tree"
	})

	resp, _, raw := doJSON(t, "GET", ts.URL+"/api/agents/repo", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("repo listing status = %d: %s", resp.StatusCode, string(raw))
	}
	var agents []engine.RepoAgent
	if err := json.Unmarshal(raw, &agents); err != nil {
		t.Fatalf("decode listing: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("entries = %d, want 2 (README excluded)", len(agents))
	}
	if agents[0].AgentID != "engineering-software-architect" ||
		agents[0].SourceURL != gh.URL+"/engineering/engineering-software-architect.md" {
		t.Fatalf("entry = %+v", agents[0])
	}
}

// TestUpdatePlanRepoAgent: a plan node sourced from the full repo is
// accepted by server-side validation (noted API.md addition) and persists
// with its source_url, so the runner can fetch it.
func TestUpdatePlanRepoAgent(t *testing.T) {
	gh := githubFixtureServer(t)
	d, mock, ts := newPipelineServer(t, time.Hour, func(s *handlers.Server) {
		s.GitHubRawBase = gh.URL
		s.GitHubTreeURL = gh.URL + "/tree"
	})
	registerPipelineRoutes(mock, loadFixture(t, "phase0_plan.json"), loadFixture(t, "agent_output.json"), loadFixture(t, "psd.md"))

	id := createSessionForRun(t, ts)
	plan := runPhase0(t, ts, id)

	modified := *plan
	repoNode := models.AgentNode{
		ID:          "game-game-designer",
		AgentID:     "game-game-designer",
		DisplayName: "Game Designer",
		SourceURL:   gh.URL + "/game-development/game-designer.md",
		Wave:        models.Wave1,
		Status:      models.StatusPending,
		Enabled:     true,
		Rationale:   "Added from the full repo.",
	}
	modified.Pipeline.Wave1 = append(append([]models.AgentNode(nil), plan.Pipeline.Wave1...), repoNode)

	resp, _, _ := doJSON(t, "PATCH", ts.URL+"/api/pipeline/"+id+"/plan", modified)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("repo-sourced patch status = %d", resp.StatusCode)
	}

	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	found := false
	for _, node := range sess.PipelinePlan.Pipeline.Wave1 {
		if node.AgentID == "game-game-designer" {
			found = true
			if node.SourceURL != gh.URL+"/game-development/game-designer.md" {
				t.Fatalf("source_url not persisted: %q", node.SourceURL)
			}
		}
	}
	if !found {
		t.Fatal("repo agent missing from persisted plan")
	}
}
