package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Config holds connection settings for the BackupSMC server.
type Config struct {
	URL   string
	Token string
}

// Reporter sends progress events to the BackupSMC server.
type Reporter struct {
	cfg    Config
	nodeID string
	client *http.Client
}

func New(cfg Config) *Reporter {
	hostname, _ := os.Hostname()
	return &Reporter{
		cfg:    cfg,
		nodeID: hostname,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *Reporter) Heartbeat(sourcePaths []string) error {
	return r.post("/api/v1/nodes/register", map[string]any{
		"node_id": r.nodeID,
		"sources": sourcePaths,
		"version": "0.5.0",
	})
}

func (r *Reporter) StartJob(jobType string) (string, error) {
	jobID := fmt.Sprintf("%s_%d", jobType, time.Now().UnixNano())
	err := r.post(fmt.Sprintf("/api/v1/jobs/%s/progress", jobID), map[string]any{
		"node_id": r.nodeID,
		"status":  "running",
		"started": time.Now().UTC(),
	})
	return jobID, err
}

func (r *Reporter) Progress(jobID string, filesTotal, filesDone int, bytesTotal, bytesDone int64, currentFile string) error {
	return r.post(fmt.Sprintf("/api/v1/jobs/%s/progress", jobID), map[string]any{
		"files_total": filesTotal,
		"files_done":  filesDone,
		"bytes_total": bytesTotal,
		"bytes_done":  bytesDone,
		"current":     currentFile,
	})
}

func (r *Reporter) Complete(jobID string, filesTotal int, bytesTotal int64) error {
	return r.post(fmt.Sprintf("/api/v1/jobs/%s/progress", jobID), map[string]any{
		"status":      "completed",
		"files_total": filesTotal,
		"bytes_total": bytesTotal,
		"finished":    time.Now().UTC(),
	})
}

func (r *Reporter) Fail(jobID, errMsg string) error {
	return r.post(fmt.Sprintf("/api/v1/jobs/%s/progress", jobID), map[string]any{
		"status":   "failed",
		"error":    errMsg,
		"finished": time.Now().UTC(),
	})
}

func (r *Reporter) post(path string, body any) error {
	if r.cfg.URL == "" {
		return nil // offline mode
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", r.cfg.URL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", r.cfg.Token)

	resp, err := r.client.Do(req)
	if err != nil {
		return err // silently ignored by engine
	}
	resp.Body.Close()
	return nil
}
