package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func Run(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = os.Getenv("USERPROFILE")
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func RunTimeout(timeout time.Duration, name string, args ...string) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Run(ctx, name, args...)
}

func PowerShell(timeout time.Duration, script string) (stdout, stderr string, err error) {
	return RunTimeout(timeout, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func Which(names ...string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

func LocalAppData(parts ...string) string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(append([]string{base}, parts...)...)
}
