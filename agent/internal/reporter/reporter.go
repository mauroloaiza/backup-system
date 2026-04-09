package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Progress represents a backup progress event sent to the server.
type Progress struct {
	JobID        string    `json:"job_id"`
	NodeID       string    `json:"node_id"`
	Status       string    `json:"status"` // running | completed | failed | warning
	FilesTotal   int64     `json:"files_total"`
	FilesDone    int64     `json:"files_done"`
	BytesTotal   int64     `json:"bytes_total"`
	BytesDone    int64     `json:"bytes_done"`
	CurrentFile  string    `json:"current_file,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Reporter sends progress updates to the BackupSMC server API.
// It rate-limits outgoing requests to avoid flooding the server.
type Reporter struct {
	serverURL  string
	apiToken   string
	minInterval time.Duration
	client     *http.Client
	log        *zap.Logger

	mu       sync.Mutex
	last     time.Time
	pending  *Progress
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// New creates a Reporter that flushes at most once per minInterval.
func New(serverURL, apiToken string, minInterval time.Duration, log *zap.Logger) *Reporter {
	r := &Reporter{
		serverURL:   serverURL,
		apiToken:    apiToken,
		minInterval: minInterval,
		client:      &http.Client{Timeout: 10 * time.Second},
		log:         log,
		stopCh:      make(chan struct{}),
	}
	r.wg.Add(1)
	go r.flusher()
	return r
}

// Update queues a progress snapshot. Thread-safe.
func (r *Reporter) Update(p Progress) {
	p.UpdatedAt = time.Now().UTC()
	r.mu.Lock()
	r.pending = &p
	r.mu.Unlock()
}

// Flush sends any pending update immediately, ignoring rate limit.
func (r *Reporter) Flush(ctx context.Context) {
	r.mu.Lock()
	p := r.pending
	r.pending = nil
	r.mu.Unlock()
	if p != nil {
		r.send(ctx, p)
	}
}

// Stop shuts down the background flusher after sending any remaining update.
func (r *Reporter) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

func (r *Reporter) flusher() {
	defer r.wg.Done()
	ticker := time.NewTicker(r.minInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			p := r.pending
			if p != nil {
				r.pending = nil
			}
			r.mu.Unlock()
			if p != nil {
				r.send(context.Background(), p)
			}
		case <-r.stopCh:
			// Final flush
			r.mu.Lock()
			p := r.pending
			r.mu.Unlock()
			if p != nil {
				r.send(context.Background(), p)
			}
			return
		}
	}
}

func (r *Reporter) send(ctx context.Context, p *Progress) {
	body, err := json.Marshal(p)
	if err != nil {
		r.log.Error("reporter: marshal progress", zap.Error(err))
		return
	}

	url := fmt.Sprintf("%s/api/v1/jobs/%s/progress", r.serverURL, p.JobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		r.log.Error("reporter: build request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", r.apiToken)

	resp, err := r.client.Do(req)
	if err != nil {
		r.log.Warn("reporter: send progress", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		r.log.Warn("reporter: server returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("job_id", p.JobID),
		)
	}
}
