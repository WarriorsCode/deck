package engine

import (
	"context"
	"os"
	"os/exec"
)

// RunShell executes a command via sh -c in the given directory.
// If env is non-nil, it replaces the process environment.
func RunShell(ctx context.Context, dir, command string, env []string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckShell runs a command and returns true if it exits 0.
func CheckShell(ctx context.Context, dir, command string, env []string) bool {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = env
	return cmd.Run() == nil
}
