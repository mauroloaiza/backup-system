//go:build windows

package vss

import (
	"fmt"
	"os/exec"
	"strings"
)

// Snapshot represents an active VSS shadow copy.
type Snapshot struct {
	ShadowID   string // e.g. {GUID}
	ShadowPath string // e.g. \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopyN
	Volume     string // e.g. C:\
}

// Create creates a VSS shadow copy of the given volume (e.g. "C:\") using
// PowerShell + WMI. Returns the shadow copy device path that can be used
// to access a consistent snapshot of the volume.
func Create(volume string) (*Snapshot, error) {
	// Normalize volume: ensure trailing backslash
	vol := strings.TrimRight(volume, `\/`) + `\`

	// PowerShell script to create VSS shadow copy and return its ID + path
	ps := fmt.Sprintf(`
$vol = '%s'
$class = [WMICLASS]"root\cimv2:win32_shadowcopy"
$result = $class.Create($vol, "ClientAccessible")
if ($result.ReturnValue -ne 0) {
    Write-Error "VSS Create failed: $($result.ReturnValue)"
    exit 1
}
$id = $result.ShadowID
$shadow = Get-WmiObject Win32_ShadowCopy | Where-Object { $_.ID -eq $id }
Write-Output "$id|$($shadow.DeviceObject)"
`, vol)

	out, err := runPS(ps)
	if err != nil {
		return nil, fmt.Errorf("vss: create shadow copy for %q: %w", volume, err)
	}

	parts := strings.SplitN(strings.TrimSpace(out), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("vss: unexpected output: %q", out)
	}

	return &Snapshot{
		ShadowID:   parts[0],
		ShadowPath: parts[1] + `\`,
		Volume:     vol,
	}, nil
}

// Delete removes the VSS shadow copy, freeing storage.
// Should always be called (defer) after backup completes.
func Delete(shadowID string) error {
	ps := fmt.Sprintf(`
$shadow = Get-WmiObject Win32_ShadowCopy | Where-Object { $_.ID -eq '%s' }
if ($shadow) { $shadow.Delete() }
`, shadowID)
	_, err := runPS(ps)
	if err != nil {
		return fmt.Errorf("vss: delete shadow %q: %w", shadowID, err)
	}
	return nil
}

// TranslatePath translates an absolute file path on the original volume to
// the equivalent path under the shadow copy device.
// e.g. "C:\Windows\file.txt" → "\\?\GLOBALROOT\Device\...\Windows\file.txt"
func (s *Snapshot) TranslatePath(origPath string) string {
	// Strip volume prefix (e.g. "C:\") and join with shadow path
	rel := strings.TrimPrefix(origPath, strings.TrimRight(s.Volume, `\`))
	rel = strings.TrimPrefix(rel, `\`)
	return s.ShadowPath + rel
}

func runPS(script string) (string, error) {
	cmd := exec.Command("powershell.exe",
		"-NonInteractive", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-Command", script,
	)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, string(ee.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
