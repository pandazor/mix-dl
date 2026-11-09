package main

import (
	"errors"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const alreadyOpenMessage = "dl-manager est déjà ouvert pour ce dossier de données."

func message(text string) {
	if path, err := exec.LookPath("zenity"); err == nil {
		_ = exec.Command(path, "--error", "--no-markup", "--title=dl-manager", "--text="+text).Run()
	}
}

func lockDataDir(dataDir string) (func(), error) {
	fd, err := unix.Open(filepath.Join(dataDir, ".instance.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		unix.Close(fd)
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, errAlreadyOpen
		}
		return nil, err
	}
	return func() { unix.Close(fd) }, nil
}
