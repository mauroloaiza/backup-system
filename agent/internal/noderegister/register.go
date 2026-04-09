// Package noderegister registers this node with the BackupSMC server and
// maintains a periodic heartbeat so the server can track agent availability.
package noderegister

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
)

const agentVersion = "0.1.0"

type payload struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Hostname     string   `json:"hostname"`
	OS           string   `json:"os"`
	AgentVersion string   `json:"agent_version"`
	SourcePaths  []string `json:"source_paths"`
}

// Register sends a single node registration to the BackupSMC server.
// Non-fatal: if the server is unreachable the agent continues normally.
func Register(ctx context.Context, serverURL, apiToken, nodeID string, sourcePaths []string) error {
	hostname, _ := os.Hostname()
	if nodeID == "" {
		nodeID = hostname
	}
	if sourcePaths == nil {
		sourcePaths = []string{}
	}

	body, _ := json.Marshal(payload{
		ID:           nodeID,
		Name:         hostname,
		Hostname:     hostname,
		OS:           runtime.GOOS,
		AgentVersion: agentVersion,
		SourcePaths:  sourcePaths,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		serverURL+"/api/v1/nodes/register", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("noderegister: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", apiToken)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("noderegister: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("noderegister: server returned %d", resp.StatusCode)
	}
	return nil
}

// StartHeartbeat registers immediately then repeats every 5 minutes.
// Runs in a background goroutine and stops when ctx is cancelled.
// No-op if serverURL is empty.
func StartHeartbeat(ctx context.Context, serverURL, apiToken, nodeID string, sourcePaths []string) {
	if serverURL == "" {
		return
	}
	go func() {
		_ = Register(ctx, serverURL, apiToken, nodeID, sourcePaths)
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = Register(ctx, serverURL, apiToken, nodeID, sourcePaths)
			}
		}
	}()
}
