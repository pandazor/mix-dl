package process

import (
	"context"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestProcessHelper(t *testing.T) {
	if os.Getenv("DLM_PROCESS_HELPER") == "child" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if os.Getenv("DLM_PROCESS_HELPER") == "parent" {
		cmd := exec.Command(os.Args[0], "-test.run=TestProcessHelper")
		cmd.Env = append(os.Environ(), "DLM_PROCESS_HELPER=child")
		if e := cmd.Start(); e != nil {
			os.Exit(2)
		}
		os.WriteFile(os.Getenv("DLM_PID_FILE"), []byte(strconv.Itoa(cmd.Process.Pid)), 0600)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
}
func TestCancellationStopsChildTree(t *testing.T) {
	file := filepath.Join(t.TempDir(), "child.pid")
	cmd := exec.Command(os.Args[0], "-test.run=TestProcessHelper")
	cmd.Env = append(os.Environ(), "DLM_PROCESS_HELPER=parent", "DLM_PID_FILE="+file)
	tree, e := Start(cmd)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- tree.Wait(ctx) }()
	defer cancel()
	var pid int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b, e := os.ReadFile(file)
		if e == nil {
			pid, _ = strconv.Atoi(string(b))
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 {
		cancel()
		<-done
		t.Fatal("child not started")
	}
	handle, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if e != nil {
		cancel()
		<-done
		t.Fatal(e)
	}
	defer windows.CloseHandle(handle)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("parent did not terminate")
	}
	status, e := windows.WaitForSingleObject(handle, 3000)
	if e != nil || status != windows.WAIT_OBJECT_0 {
		t.Fatalf("orphan child survived: %v %d", e, status)
	}
}
