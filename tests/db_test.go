package tests

import (
	"errors"
	"testing"

	"kumite/db"
	"kumite/models"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

func nodeFixture(id string, wave models.Wave) models.AgentNode {
	return models.AgentNode{
		ID:          id,
		AgentID:     id,
		DisplayName: id,
		SourceURL:   "https://example.test/" + id,
		Wave:        wave,
		Status:      models.StatusPending,
		Enabled:     true,
		Rationale:   "test rationale",
	}
}

func testPlan(sessionID string) *models.PipelinePlan {
	plan := &models.PipelinePlan{
		SessionID:   sessionID,
		ProjectName: "Field Ledger",
		InputType:   models.InputBrief,
		ProjectType: models.TypeTool,
		Context: models.PipelineContext{
			ProblemStatement: "A field data entry tool.",
			Maturity:         models.MaturityRaw,
			Domain:           "tool",
			Tags:             []string{"offline-first"},
		},
	}
	plan.Pipeline.Wave1 = []models.AgentNode{nodeFixture("arch", models.Wave1)}
	plan.Pipeline.Wave2 = []models.AgentNode{nodeFixture("pm", models.Wave2)}
	plan.Pipeline.Fixed = []models.AgentNode{nodeFixture("rc", models.Fixed)}
	return plan
}

func agentOutputFixture(agentID string) models.AgentOutput {
	return models.AgentOutput{
		AgentID:     agentID,
		DisplayName: agentID,
		Wave:        1,
		Status:      "done",
		Output: models.AgentOutputData{
			Summary: "summary text",
			Findings: []models.Finding{
				{Type: models.FindingRisk, Severity: models.SeverityCritical, Title: "sync", Body: "b"},
				{Type: models.FindingQuestion, Severity: models.SeverityLow, Title: "export", Body: "b"},
			},
			Recommendation: "rec",
			OpenQuestions:  []string{"q1"},
		},
	}
}

func TestMigrateSeedsConfigAndIsIdempotent(t *testing.T) {
	d := openTestDB(t)
	cfg, err := d.AllConfig()
	if err != nil {
		t.Fatalf("all config: %v", err)
	}
	if cfg["llm_endpoint"] != "https://ollama.com/v1" || cfg["llm_model"] != "qwen3:32b" {
		t.Fatalf("seed defaults missing: %+v", cfg)
	}
	// Re-migrate must not error or duplicate.
	if err := d.Migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	d := openTestDB(t)

	s, err := d.CreateSession("Field Ledger", "raw project material")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if s.ID == "" || s.Phase != models.PhaseIntake {
		t.Fatalf("new session wrong: %+v", s)
	}
	// raw_source is NULL exactly when intake was skipped — "" through the API.
	if s.RawSource != "" {
		t.Fatalf("raw_source should be empty, got %q", s.RawSource)
	}
	if s.RawInput != "raw project material" {
		t.Fatalf("raw_input = %q", s.RawInput)
	}

	phase := models.PhasePipelineReview
	if err := d.UpdateSession(s.ID, db.SessionPatch{Phase: &phase}); err != nil {
		t.Fatalf("update phase: %v", err)
	}
	got, err := d.GetSession(s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Phase != models.PhasePipelineReview {
		t.Fatalf("phase = %q", got.Phase)
	}

	if err := d.DeleteSession(s.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := d.GetSession(s.ID); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSessionUpdatePlanSyncsColumns(t *testing.T) {
	d := openTestDB(t)
	s, err := d.CreateSession("Field Ledger", "raw")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	plan := testPlan(s.ID)
	if err := d.SavePlan(s.ID, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	got, err := d.GetSession(s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Domain != "tool" || len(got.Tags) != 1 || got.Tags[0] != "offline-first" {
		t.Fatalf("domain/tags columns not synced: %q %+v", got.Domain, got.Tags)
	}
	if got.PipelinePlan == nil || len(got.PipelinePlan.AllNodes()) != 3 {
		t.Fatalf("plan not stored: %+v", got.PipelinePlan)
	}
}

func TestListFilters(t *testing.T) {
	d := openTestDB(t)

	s1, _ := d.CreateSession("Ledger", "in")
	s2, _ := d.CreateSession("Game", "in")
	s3, _ := d.CreateSession("NoPsd", "in")

	phase := models.PhaseComplete
	psd := "# PSD"
	if err := d.UpdateSession(s1.ID, db.SessionPatch{Phase: &phase, Psd: &psd}); err != nil {
		t.Fatalf("patch s1: %v", err)
	}
	if err := d.UpdateSession(s2.ID, db.SessionPatch{Domain: strPtr("game")}); err != nil {
		t.Fatalf("patch s2: %v", err)
	}
	if err := d.SetRawSource(s3.ID, "original"); err != nil {
		t.Fatalf("set raw source: %v", err)
	}
	if err := d.UpdateSession(s3.ID, db.SessionPatch{Tags: &[]string{"field"}}); err != nil {
		t.Fatalf("patch s3: %v", err)
	}

	byDomain, _ := d.ListSessions(db.SessionFilters{Domain: "game"})
	if len(byDomain) != 1 || byDomain[0].ID != s2.ID {
		t.Fatalf("domain filter: %+v", byDomain)
	}
	byPhase, _ := d.ListSessions(db.SessionFilters{Phase: string(models.PhaseComplete)})
	if len(byPhase) != 1 || byPhase[0].ID != s1.ID {
		t.Fatalf("phase filter: %+v", byPhase)
	}
	byTag, _ := d.ListSessions(db.SessionFilters{Tag: "field"})
	if len(byTag) != 1 || byTag[0].ID != s3.ID {
		t.Fatalf("tag filter: %+v", byTag)
	}
	hasPsd := true
	withPsd, _ := d.ListSessions(db.SessionFilters{HasPsd: &hasPsd})
	if len(withPsd) != 1 || withPsd[0].ID != s1.ID {
		t.Fatalf("has_psd filter: %+v", withPsd)
	}
	distilled := true
	withDistill, _ := d.ListSessions(db.SessionFilters{Distilled: &distilled})
	if len(withDistill) != 1 || withDistill[0].ID != s3.ID {
		t.Fatalf("distilled filter: %+v", withDistill)
	}
	distilled = false
	withoutDistill, _ := d.ListSessions(db.SessionFilters{Distilled: &distilled})
	if len(withoutDistill) != 2 {
		t.Fatalf("not-distilled filter: %+v", withoutDistill)
	}
}

func TestAppendAgentOutputAtomic(t *testing.T) {
	d := openTestDB(t)
	s, _ := d.CreateSession("Ledger", "raw")
	plan := testPlan(s.ID)
	if err := d.SavePlan(s.ID, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	// Mark running before the (mocked) LLM call, then persist the result with
	// its node status in one transaction.
	if err := d.SetNodeStatus(s.ID, "arch", models.StatusRunning); err != nil {
		t.Fatalf("set node running: %v", err)
	}
	out := agentOutputFixture("arch")
	if err := d.AppendAgentOutput(s.ID, out, "arch", models.StatusDone); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, err := d.GetSession(s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	node, _ := got.PipelinePlan.FindNode("arch")
	if node.Status != models.StatusDone {
		t.Fatalf("node status = %q, want done", node.Status)
	}
	if len(got.AgentOutputs) != 1 || got.AgentOutputs[0].AgentID != "arch" {
		t.Fatalf("outputs = %+v", got.AgentOutputs)
	}
	if got.FindingSummary.Critical != 1 || got.FindingSummary.Low != 1 {
		t.Fatalf("finding summary = %+v", got.FindingSummary)
	}
	other, _ := got.PipelinePlan.FindNode("pm")
	if other.Status != models.StatusPending {
		t.Fatalf("other nodes must stay pending, got %q", other.Status)
	}
}

func TestReconcileOrphanedRuns(t *testing.T) {
	d := openTestDB(t)

	// Session A: mid-run, one node finished (output persisted), one left running.
	a, _ := d.CreateSession("A", "raw")
	planA := testPlan(a.ID)
	d.SavePlan(a.ID, planA)
	d.SetNodeStatus(a.ID, "pm", models.StatusRunning)
	d.AppendAgentOutput(a.ID, agentOutputFixture("arch"), "arch", models.StatusDone)
	d.SetSessionPhase(a.ID, models.PhaseRunning)

	// Session B: cut during synthesis.
	b, _ := d.CreateSession("B", "raw")
	d.SetSessionPhase(b.ID, models.PhaseSynthesis)

	// Session C: idle, untouched.
	c, _ := d.CreateSession("C", "raw")

	if err := d.ReconcileOrphanedRuns(); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	ga, _ := d.GetSession(a.ID)
	if ga.Phase != models.PhaseInterrupted {
		t.Fatalf("A phase = %q, want interrupted", ga.Phase)
	}
	runningNode, _ := ga.PipelinePlan.FindNode("pm")
	if runningNode.Status != models.StatusPending {
		t.Fatalf("running node = %q, want pending", runningNode.Status)
	}
	doneNode, _ := ga.PipelinePlan.FindNode("arch")
	if doneNode.Status != models.StatusDone {
		t.Fatalf("finished node must stay done, got %q", doneNode.Status)
	}
	if len(ga.AgentOutputs) != 1 {
		t.Fatalf("partial outputs must be preserved, got %d", len(ga.AgentOutputs))
	}

	gb, _ := d.GetSession(b.ID)
	if gb.Phase != models.PhaseInterrupted {
		t.Fatalf("B phase = %q, want interrupted", gb.Phase)
	}
	gc, _ := d.GetSession(c.ID)
	if gc.Phase != models.PhaseIntake {
		t.Fatalf("C phase = %q, want intake (untouched)", gc.Phase)
	}
}

func TestConfigUpsertAndSeed(t *testing.T) {
	d := openTestDB(t)

	if err := d.UpsertConfig(map[string]string{"llm_model": "llama3:70b"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// Seeding after a user edit must NOT overwrite it.
	if err := d.SeedLLMDefaults("https://other.example/v1", "qwen3:32b", ""); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cfg, _ := d.AllConfig()
	if cfg["llm_model"] != "llama3:70b" {
		t.Fatalf("seed overwrote user value: %q", cfg["llm_model"])
	}
	if cfg["llm_endpoint"] != "https://ollama.com/v1" {
		t.Fatalf("seed should fill missing keys, got %q", cfg["llm_endpoint"])
	}

	rt, err := d.LoadRuntimeLLM()
	if err != nil {
		t.Fatalf("load runtime llm: %v", err)
	}
	if rt.Model != "llama3:70b" {
		t.Fatalf("runtime model = %q", rt.Model)
	}
}

func TestAgentCache(t *testing.T) {
	d := openTestDB(t)

	if err := d.UpsertAgentCache("arch", "https://example.test/arch", "# Architect prompt", `"abc"`); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	row, err := d.GetAgentCache("arch")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row.Content != "# Architect prompt" || row.ETag != `"abc"` || row.SourceURL != "https://example.test/arch" {
		t.Fatalf("row = %+v", row)
	}
	if err := d.TouchAgentCache("arch"); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if _, err := d.GetAgentCache("missing"); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCustomAgentsCRUD(t *testing.T) {
	d := openTestDB(t)

	created, err := d.CreateCustomAgent(models.CustomAgent{
		DisplayName:    "Slice Owner",
		RoleSummary:    "Owns PSD section 4.",
		WavePreference: 2,
		SystemPrompt:   "You are the slice owner.",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.CreatedAt == "" {
		t.Fatalf("create did not fill fields: %+v", created)
	}

	list, err := d.ListCustomAgents()
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v (%d)", err, len(list))
	}

	created.RoleSummary = "Updated summary."
	if err := d.UpdateCustomAgent(created); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := d.GetCustomAgent(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.RoleSummary != "Updated summary." {
		t.Fatalf("update not applied: %+v", got)
	}

	if err := d.DeleteCustomAgent(created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := d.GetCustomAgent(created.ID); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
