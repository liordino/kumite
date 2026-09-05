package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"kumite/db"
	"kumite/engine"
)

// handoff implements POST /api/handoff/:sessionId — generates the Dojo
// bundle from a completed session's PSD. Session must have a non-null psd.
func (s *Server) handoff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sessionId")
	sess, err := s.DB.GetSession(id)
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sess.Psd == "" {
		writeErr(w, http.StatusConflict, "session has no PSD yet (run the panel first)")
		return
	}

	cfg, err := s.llmConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	bundle, err := engine.GenerateHandoff(r.Context(), s.LLM, cfg, sess)
	if err != nil {
		var ue *engine.UnreachableError
		if errors.As(err, &ue) {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"brief_md":   bundle.BriefMD,
		"context_md": bundle.ContextMD,
	})
}
