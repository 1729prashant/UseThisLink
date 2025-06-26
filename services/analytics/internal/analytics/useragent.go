package analytics

import (
	"github.com/mssola/useragent"
)

type DeviceInfo struct {
	Browser         string
	Device          string
	OperatingSystem string
	Bot             bool
	Version         string
}

func ParseUserAgent(uaString string) DeviceInfo {
	ua := useragent.New(uaString)
	browser, version := ua.Browser()
	os := ua.OS()
	return DeviceInfo{
		Browser:         browser,
		Device:          ua.Platform(),
		OperatingSystem: os,
		Bot:             ua.Bot(),
		Version:         version,
	}
}
