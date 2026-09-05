package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"kumite/db"
	"kumite/engine"
	"kumite/models"
)

// intakeExplain implements GET /api/intake/explain — the static explanation
// of both intake paths, so every client presents the same reasoning without
// duplicating copy. Content is static; no session required.
func (s *Server) intakeExplain(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"distill": map[string]string{
			"label": "Distill first",
			"when":  "Your input is a transcript of conversations with AI models, or several files of loose brainstorming.",
			"why":   "Most text in a transcript was written by a model, not by you. Distillation separates what you decided from what a model suggested, so the panel analyzes your project rather than the model's proposals.",
			"cost":  "One extra model call. Your original is preserved.",
		},
		"skip": map[string]string{
			"label": "Use as-is",
			"when":  "Your input is a brief you wrote yourself, or an existing PSD.",
			"why":   "Distillation compresses and rephrases. When you already wrote the brief, that loses your own wording for no gain.",
			"cost":  "None.",
		},
	})
}

// runIntake implements POST /api/intake/:id — regular HTTP POST, not SSE.
// Sets phase to distilling while running, back to intake on completion or
// failure: intake does not advance the session, it only transforms the input.
// raw_source is written ONLY on success — on any error the session returns
// to intake with raw_input untouched and raw_source null.
func (s *Server) runIntake(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.DB.GetSession(id)
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess.Phase != models.PhaseIntake {
		writeErr(w, http.StatusConflict, "session is in phase "+string(sess.Phase))
		return
	}
	if sess.RawSource != "" {
		writeErr(w, http.StatusConflict, "intake already ran for this session")
		return
	}
	if sess.RawInput == "" {
		writeErr(w, http.StatusUnprocessableEntity, "raw_input is empty")
		return
	}

	original := sess.RawInput
	if err := s.DB.SetSessionPhase(id, models.PhaseDistilling); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	cfg, err := s.llmConfig()
	if err != nil {
		s.intakeFailed(w, id, err)
		return
	}
	brief, err := engine.RunIntake(r.Context(), s.LLM, cfg, original, nil)
	if err != nil {
		s.intakeFailed(w, id, err)
		return
	}

	// Success: preserve the original, replace the input, return to intake.
	if err := s.DB.SetRawSource(id, original); err != nil {
		s.intakeFailed(w, id, err)
		return
	}
	if err := s.DB.UpdateSession(id, db.SessionPatch{RawInput: &brief}); err != nil {
		s.intakeFailed(w, id, err)
		return
	}
	if err := s.DB.SetSessionPhase(id, models.PhaseIntake); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"raw_input":  brief,
		"raw_source": original,
	})
}

// intakeFailed restores the intake phase (input untouched, raw_source null)
// and maps the error: unreachable → 502, structural validation after
// retries → 500.
func (s *Server) intakeFailed(w http.ResponseWriter, id string, err error) {
	if phaseErr := s.DB.SetSessionPhase(id, models.PhaseIntake); phaseErr != nil {
		writeErr(w, http.StatusInternalServerError, phaseErr.Error())
		return
	}
	var ue *engine.UnreachableError
	if errors.As(err, &ue) {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

// revertIntake implements DELETE /api/intake/:id — restores raw_input from
// raw_source and sets raw_source back to NULL. Only valid before Phase 0 has
// run.
func (s *Server) revertIntake(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.DB.GetSession(id)
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess.Phase != models.PhaseIntake {
		writeErr(w, http.StatusConflict, "session has advanced past intake")
		return
	}
	if sess.RawSource == "" {
		writeErr(w, http.StatusNotFound, "no distillation to revert")
		return
	}

	original := sess.RawSource
	if err := s.DB.UpdateSession(id, db.SessionPatch{RawInput: &original}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.DB.ClearRawSource(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
