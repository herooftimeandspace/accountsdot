package symphony

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// RunLegacyNodeRunner calls the checked-in Node runner as a migration adapter.
// New planning and status decisions happen in Go; legacy dispatch and monitor
// side effects stay behind the existing runner until their ports are complete.
func RunLegacyNodeRunner(ctx context.Context, repoRoot string, args ...string) (map[string]any, string, error) {
	fullArgs := append([]string{"scripts/symphony_runner.mjs"}, args...)
	command := exec.CommandContext(ctx, "node", fullArgs...)
	command.Dir = repoRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, stdout.String(), fmt.Errorf("legacy node runner failed: %w: %s", err, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		return nil, stdout.String(), fmt.Errorf("decode legacy node runner JSON: %w", err)
	}
	return decoded, stdout.String(), nil
}
