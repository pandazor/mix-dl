package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCancellationStopsLinuxProcessGroup(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "pid")
	cmd := exec.Command("sh", "-c", "sleep 60 & echo $! > \"$1\"; wait", "sh", pidfile)
	tree, err := Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- tree.Wait(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	var pid int
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(pidfile)
		if err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
			if pid != 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("child did not start")
	}
	defer syscall.Kill(pid, syscall.SIGKILL)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("parent did not stop")
	}
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if i := strings.LastIndexByte(string(b), ')'); i >= 0 && len(b) > i+2 && b[i+2] == 'Z' {
			return // A zombie has stopped and awaits reaping by init.
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("child survived process-group cancellation")
}
