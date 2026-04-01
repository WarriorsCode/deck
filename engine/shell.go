package engine

import (
	"context"
	"os"
	"os/exec"
)

// RunShell executes a command via sh -c in the given directory.
func RunShell(ctx context.Context, dir, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckShell runs a command and returns true if it exits 0.
func CheckShell(ctx context.Context, dir, command string) bool {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	return cmd.Run() == nil
}
