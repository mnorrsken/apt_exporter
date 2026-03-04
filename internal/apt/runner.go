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

// Available returns true if apt-get is found in PATH (or under rootfs).
func (r *Runner) Available() bool {
	if r.rootfs == "/" || r.rootfs == "" {
		_, err := exec.LookPath("apt-get")
		return err == nil
	}
	// When using RootDir, we still invoke the local apt-get binary.
	_, err := exec.LookPath("apt-get")
	return err == nil
}

// Run executes apt-get --just-print dist-upgrade and returns stdout.
// When rootfs is not "/", it uses apt's -o RootDir option instead of chroot,
// which allows running without root privileges.
func (r *Runner) Run(ctx context.Context) (string, error) {
	args := []string{"--just-print", "dist-upgrade"}
	if r.rootfs != "/" && r.rootfs != "" {
		// Use the container's own apt method binaries (absolute path, not prefixed
		// with RootDir) to avoid GLIBC version mismatches when the host's method
		// binaries are compiled against a newer libc than the container provides.
		args = append([]string{
			"-o", "RootDir=" + r.rootfs,
			"-o", "Dir::Bin::methods=/usr/lib/apt/methods",
		}, args...)
	}
	cmd := exec.CommandContext(ctx, "apt-get", args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("apt-get failed: %w, stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
