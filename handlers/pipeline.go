package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"kumite/db"
	"kumite/engine"
	"kumite/models"
)

// Pipeline handlers. POST /run and POST /resume open an SSE stream; the run
// itself executes against a detached context because a panel run outlives the
// client that started it — a client that drops mid-run reattaches via
// GET /api/pipeline/stream/:id or reconstructs state from GET /api/sessions/:id.

// runHub tracks per-session event subscribers and in-flight runs.
type runHub struct {
	mu       sync.Mutex
	inflight map[string]bool
	subs     map[string]map[chan string]struct{}
}

func newRunHub() *runHub {
	return &runHub{
		inflight: map[string]bool{},
		subs:     map[string]map[chan string]struct{}{},
	}
}

// acquire claims the session for a run. A second run on the same session is
// rejected with 409.
func (h *runHub) acquire(sessionID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.inflight[sessionID] {
		return false
	}
	h.inflight[sessionID] = true
	return true
}

// release ends the run: subscribers' channels close, which ends their drain
// loops and the SSE responses.
func (h *runHub) release(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.inflight, sessionID)
	for ch := range h.subs[sessionID] {
		close(ch)
	}
	delete(h.subs, sessionID)
}

func (h *runHub) subscribe(sessionID string) (chan string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.inflight[sessionID] {
		return nil, false
	}
	ch := make(chan string, 256)
	if h.subs[sessionID] == nil {
		h.subs[sessionID] = map[chan string]struct{}{}
	}
	h.subs[sessionID][ch] = struct{}{}
	return ch, true
}

// emit fans one event out to every subscriber. A slow or dead subscriber
// drops events — the persisted session is the record; the stream is only the
// live tail.
func (h *runHub) emit(sessionID, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	line := "event: " + event + "\ndata: " + string(data) + "\n\n"
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[sessionID] {
		select {
		case ch <- line:
		default:
		}
	}
}

func (s *Server) hubFor() *runHub {
	if s.hub == nil {
		s.hub = newRunHub()
	}
	return s.hub
}

func (s *Server) runDeps() engine.RunDeps {
	return engine.RunDeps{
		DB:          s.DB,
		LLM:         s.LLM,
		HTTP:        s.HTTP,
		CacheTTL:    s.CacheTTL,
		Roster:      s.Roster,
		LLMOverride: s.LLMOverride,
	}
}

// mapLLMErr translates provider failures into the API's status codes:
// unreachable → 502, everything else → 500 (schema validation exhausted,
// prompt load failure).
func mapLLMErr(w http.ResponseWriter, err error) {
	var ue *engine.UnreachableError
	if errors.As(err, &ue) {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

// phase0 implements POST /api/pipeline/phase0/:id — regular POST, not SSE.
func (s *Server) phase0(w http.ResponseWriter, r *http.Request) {
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
	if sess.Phase != models.PhaseIntake && sess.Phase != models.PhasePipelineReview {
		writeErr(w, http.StatusConflict, "session is in phase "+string(sess.Phase))
		return
	}
	if strings.TrimSpace(sess.RawInput) == "" {
		writeErr(w, http.StatusUnprocessableEntity, "raw_input is empty")
		return
	}

	customAgents, err := s.DB.ListCustomAgents()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg, err := s.llmConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	plan, err := engine.RunPhase0(r.Context(), s.LLM, cfg, sess, s.Roster, customAgents)
	if err != nil {
		mapLLMErr(w, err)
		return
	}
	if err := s.DB.SavePlan(id, plan); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.DB.SetSessionPhase(id, models.PhasePipelineReview); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pipeline_plan": plan})
}

// updatePlan implements PATCH /api/pipeline/:id/plan. Server-side validation:
// the Fixed wave cannot be emptied or reordered, and every agent_id must
// resolve to a roster entry or a custom agent. Clients do not enforce this.
func (s *Server) updatePlan(w http.ResponseWriter, r *http.Request) {
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
	if sess.Phase != models.PhasePipelineReview {
		writeErr(w, http.StatusConflict, "session is in phase "+string(sess.Phase))
		return
	}

	var plan models.PipelinePlan
	if err := decodeJSON(r, &plan); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	rosterByID := map[string]models.RosterEntry{}
	for _, e := range s.Roster {
		rosterByID[e.AgentID] = e
	}
	customIDs := map[string]bool{}
	if customs, err := s.DB.ListCustomAgents(); err == nil {
		for _, c := range customs {
			customIDs[c.ID] = true
		}
	}

	resolve := func(node models.AgentNode) (models.AgentNode, bool) {
		if e, ok := rosterByID[node.AgentID]; ok {
			node.DisplayName = orDefaultStr(node.DisplayName, e.DisplayName)
			node.SourceURL = e.SourceURL
			if e.Division != "" {
				node.RoleSummary = e.DisplayName + " — " + e.Division + " division specialist"
			}
			return node, true
		}
		if customIDs[node.AgentID] {
			return node, true
		}
		// Advanced full-repo roster: a node carrying a source URL under the
		// configured raw base is accepted as-is (noted API.md addition).
		if s.isRepoSource(node.SourceURL) {
			return node, true
		}
		return models.AgentNode{}, false
	}

	plan.SessionID = id
	sanitized := models.PipelinePlan{
		SessionID:   id,
		ProjectName: orDefaultStr(plan.ProjectName, sess.ProjectName),
		InputType:   plan.InputType,
		ProjectType: plan.ProjectType,
		Context:     plan.Context,
		Pipeline: models.PipelineWaves{
			Wave1: []models.AgentNode{},
			Wave2: []models.AgentNode{},
			Fixed: []models.AgentNode{},
		},
	}

	for i := range plan.Pipeline.Wave1 {
		node, ok := resolve(plan.Pipeline.Wave1[i])
		if !ok {
			writeErr(w, http.StatusUnprocessableEntity, "unresolvable agent_id: "+node.AgentID)
			return
		}
		if node.Wave != models.Wave1 {
			writeErr(w, http.StatusUnprocessableEntity, "wave 1 node with wave "+fmt.Sprint(int(node.Wave))+": "+node.AgentID)
			return
		}
		node.ID = node.AgentID
		node.Status = models.StatusPending
		sanitized.Pipeline.Wave1 = append(sanitized.Pipeline.Wave1, node)
	}
	for i := range plan.Pipeline.Wave2 {
		node, ok := resolve(plan.Pipeline.Wave2[i])
		if !ok {
			writeErr(w, http.StatusUnprocessableEntity, "unresolvable agent_id: "+node.AgentID)
			return
		}
		if node.Wave != models.Wave2 {
			writeErr(w, http.StatusUnprocessableEntity, "wave 2 node with wave "+fmt.Sprint(int(node.Wave))+": "+node.AgentID)
			return
		}
		node.ID = node.AgentID
		node.Status = models.StatusPending
		sanitized.Pipeline.Wave2 = append(sanitized.Pipeline.Wave2, node)
	}

	// The Fixed wave cannot be emptied, disabled, or reordered: it must be
	// exactly the roster's fixed agents, enabled, in the fixed wave.
	var fixedIDs []string
	for _, e := range s.Roster {
		if e.Fixed {
			fixedIDs = append(fixedIDs, e.AgentID)
		}
	}
	gotFixed := map[string]bool{}
	for i := range plan.Pipeline.Fixed {
		node, ok := resolve(plan.Pipeline.Fixed[i])
		if !ok {
			writeErr(w, http.StatusUnprocessableEntity, "unresolvable agent_id: "+node.AgentID)
			return
		}
		entry := rosterByID[node.AgentID]
		if !entry.Fixed || node.Wave != models.Fixed || !node.Enabled {
			writeErr(w, http.StatusConflict, "the Fixed wave cannot be emptied, disabled, or reordered")
			return
		}
		gotFixed[node.AgentID] = true
		node.ID = node.AgentID
		node.Status = models.StatusPending
		node.Enabled = true
		sanitized.Pipeline.Fixed = append(sanitized.Pipeline.Fixed, node)
	}
	for _, fid := range fixedIDs {
		if !gotFixed[fid] {
			writeErr(w, http.StatusConflict, "the Fixed wave cannot be emptied, disabled, or reordered")
			return
		}
	}
	if len(sanitized.Pipeline.Wave1)+len(sanitized.Pipeline.Wave2) == 0 {
		writeErr(w, http.StatusUnprocessableEntity, "plan contains no specialists")
		return
	}

	if err := s.DB.SavePlan(id, &sanitized); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func orDefaultStr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// run implements POST /api/pipeline/run/:id — opens the SSE stream and
// executes the pipeline. The run detaches from the request context: it
// outlives the client.
func (s *Server) run(w http.ResponseWriter, r *http.Request) {
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
	if sess.Phase != models.PhasePipelineReview {
		writeErr(w, http.StatusConflict, "session is in phase "+string(sess.Phase))
		return
	}

	// Optional JSON body: {"pause_on_fail": true} — the policy persists on
	// the plan so a resume keeps it. Empty body = keep the current policy.
	var opts struct {
		PauseOnFail *bool `json:"pause_on_fail"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &opts); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
			return
		}
		if opts.PauseOnFail != nil && sess.PipelinePlan != nil {
			sess.PipelinePlan.PauseOnFail = *opts.PauseOnFail
			if err := s.DB.SavePlan(id, sess.PipelinePlan); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	s.startRun(w, r, id, false)
}

// resume implements POST /api/pipeline/resume/:id. It walks the persisted
// plan in wave order, skipping nodes already done or skipped, and emits
// run_resumed first so a client can render the recovered state without a
// separate code path.
func (s *Server) resume(w http.ResponseWriter, r *http.Request) {
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
	if sess.Phase != models.PhaseInterrupted {
		writeErr(w, http.StatusConflict, "session is in phase "+string(sess.Phase))
		return
	}
	s.startRun(w, r, id, true)
}

func (s *Server) startRun(w http.ResponseWriter, r *http.Request, id string, resume bool) {
	hub := s.hubFor()
	if !hub.acquire(id) {
		writeErr(w, http.StatusConflict, "a run is already in flight for this session")
		return
	}

	flusher, canFlush := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if canFlush {
		flusher.Flush()
	}

	sub, ok := hub.subscribe(id)
	if !ok {
		hub.release(id)
		return
	}

	if resume {
		// Re-read for the freshest plan; emit the resume preamble.
		if sess, err := s.DB.GetSession(id); err == nil && sess.PipelinePlan != nil {
			skipping := engine.SkippedAgentIDs(sess.PipelinePlan)
			hub.emit(id, "run_resumed", map[string]any{
				"session_id": id,
				"skipping":   skipping,
				"remaining":  engine.RemainingAgents(sess.PipelinePlan),
			})
		}
	}

	// The run detaches from the request context — it outlives this client.
	go func() {
		defer hub.release(id)
		sess, err := s.DB.GetSession(id)
		if err != nil {
			hub.emit(id, "pipeline_error", map[string]any{"error": err.Error()})
			return
		}
		emit := func(event string, payload any) { hub.emit(id, event, payload) }
		if err := engine.RunPipeline(context.Background(), s.runDeps(), sess, emit); err != nil {
			hub.emit(id, "pipeline_error", map[string]any{"error": err.Error()})
		}
	}()

	for {
		select {
		case ev, ok := <-sub:
			if !ok {
				return // run finished; stream closes
			}
			if _, writeErr := fmt.Fprint(w, ev); writeErr != nil {
				return // client gone; the run continues server-side
			}
			if canFlush {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return // client disconnect; the run continues server-side
		}
	}
}

// stream implements GET /api/pipeline/stream/:id — attach to a run already in
// flight. Events are not replayed: the persisted session is the record.
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := s.DB.GetSession(id); err == db.ErrNotFound {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	hub := s.hubFor()
	sub, ok := hub.subscribe(id)
	if !ok {
		writeErr(w, http.StatusConflict, "no run in flight for this session")
		return
	}

	flusher, canFlush := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if canFlush {
		flusher.Flush()
	}
	for {
		select {
		case ev, ok := <-sub:
			if !ok {
				return
			}
			if _, writeErr := fmt.Fprint(w, ev); writeErr != nil {
				return
			}
			if canFlush {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}
