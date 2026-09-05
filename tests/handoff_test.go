package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"kumite/db"
	"kumite/models"
)

func TestHandoffHTTP(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	mock.Register(ContentContains("INVOCATION: Handoff"), func(MockRequest) string {
		return loadFixture(t, "handoff_bundle.md")
	})

	// A completed session: PSD present, phase complete.
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "Field Ledger", "raw_input": "raw",
	})
	id := m["id"].(string)
	psd := loadFixture(t, "psd.md")
	psdCopy := psd
	phase := models.PhaseComplete
	if err := d.UpdateSession(id, db.SessionPatch{Psd: &psdCopy, Phase: &phase}); err != nil {
		t.Fatalf("patch session: %v", err)
	}

	resp, m, raw := doJSON(t, "POST", ts.URL+"/api/handoff/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("handoff status = %d: %s", resp.StatusCode, string(raw))
	}
	brief, _ := m["brief_md"].(string)
	contextSeed, _ := m["context_md"].(string)
	if !strings.Contains(brief, "# Field Ledger — Project Brief") {
		t.Fatalf("brief_md missing or wrong: %.80s", brief)
	}
	if !strings.Contains(contextSeed, "## Glossary") || !strings.Contains(contextSeed, "## Non-Goals") || !strings.Contains(contextSeed, "## Decisions") {
		t.Fatalf("context seed missing sections: %.120s", contextSeed)
	}

	// Unknown session → 404.
	if resp, _, _ = doJSON(t, "POST", ts.URL+"/api/handoff/no-such-session", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown session status = %d, want 404", resp.StatusCode)
	}

	// Session without a PSD → 409.
	resp, m2, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "No PSD", "raw_input": "raw",
	})
	if resp, _, _ = doJSON(t, "POST", ts.URL+"/api/handoff/"+m2["id"].(string), nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("no-PSD handoff status = %d, want 409", resp.StatusCode)
	}
}
