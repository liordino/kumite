package handlers

import (
	"net/http"

	"kumite/engine"
)

// getConfig implements GET /api/config. The API key is returned masked —
// never in the clear.
func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	cfg, err := s.DB.AllConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg["llm_api_key"] = engine.MaskAPIKey(cfg["llm_api_key"])
	writeJSON(w, http.StatusOK, cfg)
}

// putConfig implements PUT /api/config — partial upsert that takes effect on
// the next LLM call. A request carrying the masked key leaves the stored key
// untouched, so a client that round-trips the GET body cannot clobber the
// real value.
func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := decodeJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if v, ok := body["llm_api_key"]; ok && engine.MaskAPIKey(v) == v && v != "" {
		delete(body, "llm_api_key")
	}
	if len(body) > 0 {
		if err := s.DB.UpsertConfig(body); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}