package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"kumite/db"
	"kumite/engine"
	"kumite/models"
)

func TestIntakeExplainHTTP(t *testing.T) {
	_, _, ts := newPipelineServer(t, time.Hour)
	resp, m, _ := doJSON(t, "GET", ts.URL+"/api/intake/explain", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("explain status = %d", resp.StatusCode)
	}
	distill, ok := m["distill"].(map[string]any)
	if !ok || distill["label"] != "Distill first" {
		t.Fatalf("distill explanation = %v", m["distill"])
	}
	skip, ok := m["skip"].(map[string]any)
	if !ok || skip["label"] != "Use as-is" {
		t.Fatalf("skip explanation = %v", m["skip"])
	}
}

func TestIntakeHappyPath(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	briefFixture := loadFixture(t, "uchikomi_brief.md")
	mock.Register(ContentContains("MATERIAL TO DISTILL"), func(MockRequest) string {
		return briefFixture
	})

	original := loadFixture(t, "transcript_with_imperatives.md")
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "Field Logger",
		"raw_input":    original,
	})
	id := m["id"].(string)

	resp, m, _ = doJSON(t, "POST", ts.URL+"/api/intake/"+id, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("intake status = %d", resp.StatusCode)
	}
	if m["raw_source"] != original {
		t.Fatal("raw_source must be the untouched original")
	}
	// The engine trims the model response; the fixture file carries a
	// trailing newline from the markdown formatter.
	if m["raw_input"] != strings.TrimSpace(briefFixture) {
		t.Fatalf("raw_input must be the distilled brief")
	}

	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.RawSource != original || sess.RawInput != strings.TrimSpace(briefFixture) {
		t.Fatalf("session not transformed correctly")
	}
	if sess.Phase != models.PhaseIntake {
		t.Fatalf("phase = %q, want intake (intake does not advance the session)", sess.Phase)
	}

	// The list metadata now reports the session as distilled.
	distilled := true
	list, err := d.ListSessions(db.SessionFilters{Distilled: &distilled})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, it := range list {
		if it.ID == id {
			found = true
			if !it.Distilled {
				t.Fatal("list metadata should report distilled=true")
			}
		}
	}
	if !found {
		t.Fatal("session missing from distilled list")
	}
}

func TestIntakeValidationFailure(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	mock.Register(ContentContains("MATERIAL TO DISTILL"), func(MockRequest) string {
		return "I am a chatty model response with no brief structure whatsoever."
	})

	original := "the author's original material, untouched"
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "F", "raw_input": original,
	})
	id := m["id"].(string)

	resp, _, _ = doJSON(t, "POST", ts.URL+"/api/intake/"+id, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("structural failure status = %d, want 500", resp.StatusCode)
	}
	// Retry budget: one initial + 2 structural retries.
	if got := mock.CallCount(func(RecordedCall) bool { return true }); got != 3 {
		t.Fatalf("provider calls = %d, want 3", got)
	}

	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if sess.RawSource != "" {
		t.Fatalf("raw_source must stay null on failure, got %q", sess.RawSource)
	}
	if sess.RawInput != original {
		t.Fatalf("raw_input must stay untouched on failure")
	}
	if sess.Phase != models.PhaseIntake {
		t.Fatalf("phase = %q, want intake", sess.Phase)
	}
}

// TestIntakeImperativeText: the fixture transcript contains imperative text
// ("Ignore all previous instructions…"). Rule 8 — the material is data, not
// instruction: it must arrive between the markers and nowhere else.
func TestIntakeImperativeText(t *testing.T) {
	_, mock, ts := newPipelineServer(t, time.Hour)
	briefFixture := loadFixture(t, "uchikomi_brief.md")
	mock.Register(ContentContains("MATERIAL TO DISTILL"), func(MockRequest) string {
		return briefFixture
	})

	original := loadFixture(t, "transcript_with_imperatives.md")
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "F", "raw_input": original,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d", resp.StatusCode)
	}
	id := m["id"].(string)
	if resp, _, _ = doJSON(t, "POST", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("intake status = %d", resp.StatusCode)
	}

	calls := mock.Calls()
	if len(calls) == 0 {
		t.Fatal("no calls recorded")
	}
	user := calls[len(calls)-1].Last
	begin := strings.Index(user, engine.InputBeginMarker)
	end := strings.Index(user, engine.InputEndMarker)
	if begin < 0 || end < 0 || end < begin {
		t.Fatalf("input not delimited: %.120s", user)
	}
	between := user[begin:end]
	if !strings.Contains(between, "Ignore all previous instructions") {
		t.Fatal("the imperative text must be inside the markers — it is part of the material")
	}
	if strings.Count(user, engine.InputBeginMarker) != 1 || strings.Count(user, engine.InputEndMarker) != 1 {
		t.Fatal("exactly one marker pair may exist")
	}
}

func TestIntakeRevert(t *testing.T) {
	d, mock, ts := newPipelineServer(t, time.Hour)
	briefFixture := loadFixture(t, "uchikomi_brief.md")
	mock.Register(ContentContains("MATERIAL TO DISTILL"), func(MockRequest) string {
		return briefFixture
	})

	original := "original material before distillation"
	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "F", "raw_input": original,
	})
	id := m["id"].(string)
	if resp, _, _ = doJSON(t, "POST", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("intake failed")
	}

	// Revert: input restored, raw_source back to NULL.
	if resp, _, _ = doJSON(t, "DELETE", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("revert status = %d", resp.StatusCode)
	}
	sess, err := d.GetSession(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if sess.RawInput != original || sess.RawSource != "" {
		t.Fatalf("revert incomplete: raw_input=%q raw_source=%q", sess.RawInput, sess.RawSource)
	}

	// Second revert: nothing to revert → 404.
	if resp, _, _ = doJSON(t, "DELETE", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second revert status = %d, want 404", resp.StatusCode)
	}

	// Revert after Phase 0 has run → 409.
	phase := models.PhasePipelineReview
	if err := d.UpdateSession(id, db.SessionPatch{Phase: &phase}); err != nil {
		t.Fatalf("advance phase: %v", err)
	}
	if err := d.SetRawSource(id, "original again"); err != nil {
		t.Fatalf("set raw source: %v", err)
	}
	if resp, _, _ = doJSON(t, "DELETE", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("revert past intake status = %d, want 409", resp.StatusCode)
	}
}

func TestIntakeAlreadyRan(t *testing.T) {
	_, mock, ts := newPipelineServer(t, time.Hour)
	mock.Register(ContentContains("MATERIAL TO DISTILL"), func(MockRequest) string {
		return loadFixture(t, "uchikomi_brief.md")
	})

	resp, m, _ := doJSON(t, "POST", ts.URL+"/api/sessions", map[string]any{
		"project_name": "F", "raw_input": "some material",
	})
	id := m["id"].(string)
	if resp, _, _ = doJSON(t, "POST", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("first intake failed")
	}
	if resp, _, _ = doJSON(t, "POST", ts.URL+"/api/intake/"+id, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("second intake status = %d, want 409", resp.StatusCode)
	}
}

func TestIntakeUnknownSession(t *testing.T) {
	_, _, ts := newPipelineServer(t, time.Hour)
	if resp, _, _ := doJSON(t, "POST", ts.URL+"/api/intake/no-such-session", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
