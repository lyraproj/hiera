// +build windows

package session

import (
	"golang.org/x/sys/windows/registry"
	"strconv"
)

// getDefaultPluginTransport returns the plugin transport method to use.
// For Windows this can be unix if the OS build is 17063 or above,
// otherwise tcp is used
// https://devblogs.microsoft.com/commandline/af_unix-comes-to-windows/
func getDefaultPluginTransport() string {
	if isBuild17063() {
		return pluginTransportUnix
	}
	return pluginTransportTCP
}

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
