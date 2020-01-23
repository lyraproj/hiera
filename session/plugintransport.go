// +build !windows

package session

func getDefaultPluginTransport() string {
	return pluginTransportUnix
}
