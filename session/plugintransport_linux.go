// +build linux

package session

func getDefaultPluginTransport() string {
	return pluginTransportUnix
}
