//go:build linux

package updater

import "syscall"

func getSysProcAttrWindows() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
