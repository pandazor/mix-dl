package process

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"os/exec"
	"syscall"
	"unsafe"
)

type Tree struct {
	cmd *exec.Cmd
	job windows.Handle
}

func Start(cmd *exec.Cmd) (*Tree, error) {
	job, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return nil, e
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); e != nil {
		windows.CloseHandle(job)
		return nil, e
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_SUSPENDED | windows.CREATE_NO_WINDOW}
	if e = cmd.Start(); e != nil {
		windows.CloseHandle(job)
		return nil, e
	}
	fail := func(err error) (*Tree, error) {
		cmd.Process.Kill()
		cmd.Wait()
		windows.CloseHandle(job)
		return nil, err
	}
	proc, e := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SUSPEND_RESUME, false, uint32(cmd.Process.Pid))
	if e != nil {
		return fail(e)
	}
	defer windows.CloseHandle(proc)
	if e = windows.AssignProcessToJobObject(job, proc); e != nil {
		return fail(e)
	}
	status, _, _ := windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess").Call(uintptr(proc))
	if status != 0 {
		return fail(fmt.Errorf("NtResumeProcess: 0x%x", status))
	}
	return &Tree{cmd, job}, nil
}
func (t *Tree) Wait(ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- t.cmd.Wait() }()
	select {
	case e := <-done:
		windows.CloseHandle(t.job)
		return e
	case <-ctx.Done():
		windows.TerminateJobObject(t.job, 1)
		e := <-done
		_ = e
		windows.CloseHandle(t.job)
		return ctx.Err()
	}
}

func ProbeOutput(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}
