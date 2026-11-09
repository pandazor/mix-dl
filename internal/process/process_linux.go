package process

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type Tree struct {
	cmd *exec.Cmd
}

func Start(cmd *exec.Cmd) (*Tree, error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Tree{cmd}, nil
}

func (t *Tree) Wait(ctx context.Context) error {
	pid := t.cmd.Process.Pid
	var mu sync.Mutex
	stopped := false
	stopGroup := func() {
		mu.Lock()
		defer mu.Unlock()
		if !stopped {
			unix.Kill(-pid, unix.SIGKILL)
			stopped = true
		}
	}
	done := make(chan error, 1)
	go func() {
		// Keep the leader unreaped until the group is signaled, so its ID
		// cannot be reused by an unrelated process group.
		var info unix.Siginfo
		err := unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
		for errors.Is(err, unix.EINTR) {
			err = unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
		}
		stopGroup()
		waitErr := t.cmd.Wait()
		if err != nil {
			waitErr = err
		}
		done <- waitErr
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		stopGroup()
		<-done
		return ctx.Err()
	}
}

func ProbeOutput(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := exec.Command(path, args...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	tree, err := Start(cmd)
	if err != nil {
		return nil, err
	}
	err = tree.Wait(ctx)
	return out.Bytes(), err
}
