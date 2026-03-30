package profile

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load pulls the profile OCI image identified by ref and returns the parsed Profile.
// Placeholders of the form ${KEY} in config values are resolved from userConfig.
// Returns *MissingConfigError if any placeholder has no supplied value.
func Load(ctx context.Context, ref string, userConfig map[string]string) (*Profile, error) {
	// Derive a stable temp file name from the ref hash so concurrent loads of
	// different profiles don't collide.
	h := sha256.Sum256([]byte(ref))
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("profile-gateway-%x.yaml", h[:8]))
	containerName := fmt.Sprintf("profile-gateway-tmp-%x", h[:8])

	// 1. docker pull
	if out, err := exec.CommandContext(ctx, "docker", "pull", ref).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker pull %s: %w\n%s", ref, err, out)
	}

	// Ensure the temp container is removed even if we fail partway through.
	defer func() {
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
	}()

	// 2. docker create
	if out, err := exec.CommandContext(ctx, "docker", "create", "--name", containerName, ref, "x").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker create %s: %w\n%s", ref, err, out)
	}

	// 3. docker cp
	if out, err := exec.CommandContext(ctx, "docker", "cp", containerName+":/profile.yaml", tmpPath).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker cp profile.yaml from %s: %w\n%s", ref, err, out)
	}

	// 4. docker rm happens in the deferred cleanup above.

	// 5. Parse yaml.
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("read profile.yaml: %w", err)
	}
	_ = os.Remove(tmpPath)

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile.yaml: %w", err)
	}
	if err := ResolvePlaceholders(&p, userConfig); err != nil {
		return nil, err
	}
	return &p, nil
}
