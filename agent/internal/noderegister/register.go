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
	"strconv"
	"time"

	"github.com/smcsoluciones/backup-system/agent/internal/config"
)

const agentVersion = "0.4.0"

// Destination is the payload-shape for a single configured destination that
// the server exposes on GET /api/v1/destinations.
type Destination struct {
	Type    string            `json:"type"`              // local | s3 | sftp
	Target  string            `json:"target"`            // human-readable
	Details map[string]string `json:"details,omitempty"` // provider-specific extras
}

type payload struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Hostname     string        `json:"hostname"`
	OS           string        `json:"os"`
	AgentVersion string        `json:"agent_version"`
	SourcePaths  []string      `json:"source_paths"`
	Destinations []Destination `json:"destinations,omitempty"`
}

// BuildDestinations converts the agent's destination config into the payload
// format expected by the server. Secrets (passwords, key-file contents) are
// never included — only structural info useful for the UI.
func BuildDestinations(dc config.DestinationConfig) []Destination {
	t := dc.Type
	if t == "" {
		t = "local"
	}
	switch t {
	case "local":
		if dc.LocalPath == "" {
			return nil
		}
		return []Destination{{
			Type:   "local",
			Target: dc.LocalPath,
		}}
	case "s3":
		if dc.S3Bucket == "" {
			return nil
		}
		details := map[string]string{"bucket": dc.S3Bucket}
		if dc.S3Region != "" {
			details["region"] = dc.S3Region
		}
		if dc.S3Prefix != "" {
			details["prefix"] = dc.S3Prefix
		}
		target := "s3://" + dc.S3Bucket
		if dc.S3Prefix != "" {
			target += "/" + dc.S3Prefix
		}
		return []Destination{{Type: "s3", Target: target, Details: details}}
	case "sftp":
		if dc.SFTPHost == "" {
			return nil
		}
		port := dc.SFTPPort
		if port == 0 {
			port = 22
		}
		details := map[string]string{
			"host": dc.SFTPHost,
			"port": strconv.Itoa(port),
			"user": dc.SFTPUser,
			"path": dc.SFTPPath,
		}
		target := fmt.Sprintf("sftp://%s@%s:%d%s", dc.SFTPUser, dc.SFTPHost, port, dc.SFTPPath)
		return []Destination{{Type: "sftp", Target: target, Details: details}}
	}
	return nil
}

// Register sends a single node registration to the BackupSMC server.
// Non-fatal: if the server is unreachable the agent continues normally.
func Register(ctx context.Context, serverURL, apiToken, nodeID string, sourcePaths []string, destinations []Destination) error {
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
		Destinations: destinations,
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
func StartHeartbeat(ctx context.Context, serverURL, apiToken, nodeID string, sourcePaths []string, destinations []Destination) {
	if serverURL == "" {
		return
	}
	go func() {
		_ = Register(ctx, serverURL, apiToken, nodeID, sourcePaths, destinations)
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = Register(ctx, serverURL, apiToken, nodeID, sourcePaths, destinations)
			}
		}
	}()
}
