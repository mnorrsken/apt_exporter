package apt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Runner executes apt-get and returns its stdout.
type Runner struct {
	rootfs string
}

// NewRunner creates a new Runner. rootfs is the path to the root filesystem
// (use "/" for the local system, "/host" for container with host mount).
func NewRunner(rootfs string) *Runner {
	return &Runner{rootfs: rootfs}
}

// RootFS returns the configured root filesystem path.
func (r *Runner) RootFS() string {
	return r.rootfs
}

// Run executes apt-get --just-print dist-upgrade and returns stdout.
func (r *Runner) Run(ctx context.Context) (string, error) {
	var cmd *exec.Cmd
	if r.rootfs == "/" || r.rootfs == "" {
		cmd = exec.CommandContext(ctx, "apt-get", "--just-print", "dist-upgrade")
	} else {
		cmd = exec.CommandContext(ctx, "chroot", r.rootfs, "apt-get", "--just-print", "dist-upgrade")
	}
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("apt-get failed: %w, stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
