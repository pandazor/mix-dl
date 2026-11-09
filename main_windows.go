package main

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const alreadyOpenMessage = "dl-manager est déjà ouvert. Retrouvez-le dans la zone de notification."

func message(text string) {
	p, _ := windows.UTF16PtrFromString(text)
	title, _ := windows.UTF16PtrFromString("dl-manager")
	windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(title)), 0x10)
}

func lockDataDir(dataDir string) (func(), error) {
	name, _ := windows.UTF16PtrFromString("Local\\dl-manager-" + strings.NewReplacer("\\", "_", ":", "_").Replace(strings.ToLower(dataDir)))
	mutex, err := windows.CreateMutex(nil, true, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(mutex)
		return nil, errAlreadyOpen
	}
	if err != nil {
		return nil, err
	}
	return func() { windows.CloseHandle(mutex) }, nil
}
