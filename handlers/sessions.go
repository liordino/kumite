package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"kumite/db"
	"kumite/models"
)

// listSessions implements GET /api/sessions with its query filters.
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	f := db.SessionFilters{
		Domain: r.URL.Query().Get("domain"),
		Phase:  r.URL.Query().Get("phase"),
		Tag:    r.URL.Query().Get("tag"),
	}
	if v := r.URL.Query().Get("has_psd"); v != "" {
		b := v == "true"
		f.HasPsd = &b
	}
	if v := r.URL.Query().Get("distilled"); v != "" {
		b := v == "true"
		f.Distilled = &b
	}
	items, err := s.DB.ListSessions(f)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []models.SessionListItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

// createSession implements POST /api/sessions. The submitted material goes to
// raw_input; raw_source stays NULL — its absence is the record that intake
// has not run.
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectName string `json:"project_name"`
		RawInput    string `json:"raw_input"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	sess, err := s.DB.CreateSession(body.ProjectName, body.RawInput)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": sess.ID})
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.DB.GetSession(chi.URLParam(r, "id"))
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// patchSession implements PATCH /api/sessions/:id — partial update of the
// permitted fields. raw_source is NOT writable through this route: it is set
// only by the intake endpoint, because its NULL state is a record of which
// path the session took. Complex fields arrive as objects and are serialized
// inside the db package.
func (s *Server) patchSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		ProjectName  *string               `json:"project_name"`
		Phase        *models.SessionPhase  `json:"phase"`
		RawInput     *string               `json:"raw_input"`
		PipelinePlan *models.PipelinePlan  `json:"pipeline_plan"`
		AgentOutputs *[]models.AgentOutput `json:"agent_outputs"`
		Psd          *string               `json:"psd"`
		Tags         *[]string             `json:"tags"`
		Domain       *string               `json:"domain"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	patch := db.SessionPatch{
		ProjectName:  body.ProjectName,
		Phase:        body.Phase,
		RawInput:     body.RawInput,
		PipelinePlan: body.PipelinePlan,
		AgentOutputs: body.AgentOutputs,
		Psd:          body.Psd,
		Tags:         body.Tags,
		Domain:       body.Domain,
	}
	if err := s.DB.UpdateSession(id, patch); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	err := s.DB.DeleteSession(chi.URLParam(r, "id"))
	if err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}