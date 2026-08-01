package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloud-engineer-test/internal/config"
	"github.com/cloud-engineer-test/internal/model"
	"github.com/cloud-engineer-test/internal/s3"
	"github.com/cloud-engineer-test/internal/store"
	"github.com/google/uuid"
)

type server struct {
	cfg   *config.Config
	store *store.Store
	s3    *s3.Client
}

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" || cfg.Bucket == "" {
		log.Fatal("DATABASE_URL and MEDIA_BUCKET are required")
	}

	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	media, err := s3.New(ctx, cfg.Bucket, cfg.Region)
	if err != nil {
		log.Fatalf("s3: %v", err)
	}

	s := &server{cfg: cfg, store: st, s3: media}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /upload", s.handleUpload)
	mux.HandleFunc("POST /sessions/{id}/apply", s.handleApply)
	mux.HandleFunc("GET /sessions/{id}/status", s.handleStatus)
	mux.HandleFunc("GET /sessions/{id}/download", s.handleDownload)

	addr := ":" + cfg.Port
	log.Printf("api listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".mp4"
	}
	ext = strings.TrimPrefix(ext, ".")

	id := uuid.NewString()
	sess, err := s.store.CreateSession(r.Context(), id, ext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	key := s3.Key(id, sess.CurrentVersion, ext)
	if err := s.s3.Upload(r.Context(), key, header.Header.Get("Content-Type"), file); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"session_id": id,
		"version":    sess.CurrentVersion,
	})
}

func (s *server) handleApply(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Operation model.OperationType `json:"operation"`
		Params    map[string]any      `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !body.Operation.Valid() {
		writeError(w, http.StatusBadRequest, "unsupported operation")
		return
	}

	sess, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	job, err := s.store.CreateJob(r.Context(), id, body.Operation, body.Params, sess.CurrentVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id":         job.ID,
		"input_version":  job.InputVersion,
		"output_version": job.OutputVersion,
	})
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	job, err := s.store.LatestJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]any{
		"session_id":      sess.ID,
		"current_version": sess.CurrentVersion,
		"extension":       sess.Extension,
		"job":             job,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	key := s3.Key(id, sess.CurrentVersion, sess.Extension)
	url, err := s.s3.PresignedURL(r.Context(), key, s.cfg.PresignTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"download_url": url,
		"version":      sess.CurrentVersion,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
