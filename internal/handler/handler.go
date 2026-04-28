package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aoideee/quiz-job-queue/internal/db"
)

// Handler holds the store dependency.
type Handler struct {
	store *db.Store
}

func New(store *db.Store) *Handler {
	return &Handler{store: store}
}

// --- POST /jobs ---
// Body: { "payload": { ... } }

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(body.Payload) == 0 {
		writeErr(w, http.StatusBadRequest, "payload is required")
		return
	}

	job, err := h.store.CreateJob(r.Context(), body.Payload)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to create job")
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

// --- GET /jobs/{id} ---

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.store.GetJob(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to fetch job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// --- PATCH /jobs/{id} ---
// Body: { "status": "done", "progress": 100, "result": {...}, "error_msg": "..." }
// All fields are optional.

func (h *Handler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Status   *string         `json:"status"`
		Progress *int            `json:"progress"`
		Result   json.RawMessage `json:"result"`
		ErrorMsg *string         `json:"error_msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	params := db.UpdateParams{
		Status:   body.Status,
		Progress: body.Progress,
		Result:   body.Result,
		ErrorMsg: body.ErrorMsg,
	}

	job, err := h.store.UpdateJob(r.Context(), id, params)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to update job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// --- GET /jobs/next ---
// Atomically claims and returns the next pending job (for workers).

func (h *Handler) ClaimNextJob(w http.ResponseWriter, r *http.Request) {
	job, err := h.store.ClaimNextJob(r.Context())
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to claim job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// --- GET /jobs/stream ---
// Server-Sent Events: polls the DB every 2s for jobs updated since last check
// and pushes them as "data: <json>\n\n" events.

func (h *Handler) StreamUpdates(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send a comment to keep the connection alive immediately.
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	cursor := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case t := <-ticker.C:
			jobs, err := h.store.JobsSince(context.Background(), cursor)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
				continue
			}
			cursor = t
			for _, job := range jobs {
				data, _ := json.Marshal(job)
				fmt.Fprintf(w, "data: %s\n\n", data)
			}
			if len(jobs) > 0 {
				flusher.Flush()
			}
		}
	}
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
