package analytics

import (
	"github.com/mssola/useragent"
)

type DeviceInfo struct {
	Browser         string
	Device          string
	OperatingSystem string
	Version         string
	Mobile          bool
	Bot             bool
}

func ParseUserAgent(userAgentString string) DeviceInfo {
	ua := useragent.New(userAgentString)
	info := DeviceInfo{
		Mobile: ua.Mobile(),
		Bot:    ua.Bot(),
	}

	// Get browser name and version
	name, version := ua.Browser()
	info.Browser = name
	info.Version = version

	// Get OS
	info.OperatingSystem = ua.OS()

	// Determine device type based on mobile flag and OS
	switch {
	case info.Bot:
		info.Device = "Bot"
	case info.Mobile:
		if ua.Platform() == "iPad" {
			info.Device = "Tablet"
		} else {
			info.Device = "Mobile"
		}
	default:
		info.Device = "Desktop"
	}

	return info
}
