// +build windows

package session

import (
	"golang.org/x/sys/windows/registry"
	"strconv"
)

// isBuild17063 gets the Windows build number from the registry
func isBuild17063() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.READ)
	if err != nil {
		return false
	}
	defer k.Close()
	s, _, err := k.GetStringValue("CurrentBuild")
	if err != nil {
		return false
	}
	ver, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return ver >= 17063
}
