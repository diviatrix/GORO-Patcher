//go:build windows

package updater

import "syscall"

func getSysProcAttrWindows() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000,
	}
}
