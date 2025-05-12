package analytics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeoInfo struct {
	City    string
	Country string
}

func GetLocationFromIP(ipAddr string) (GeoInfo, error) {
	// Use free IP geolocation API
	resp, err := http.Get(fmt.Sprintf("https://ipapi.co/%s/json/", ipAddr))
	if err != nil {
		return GeoInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeoInfo{}, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return GeoInfo{}, err
	}

	info := GeoInfo{
		City:    fmt.Sprintf("%v", result["city"]),
		Country: fmt.Sprintf("%v", result["country_name"]),
	}

	return info, nil
}
