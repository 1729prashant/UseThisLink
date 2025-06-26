package analytics

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type GeoInfo struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

func GetLocationFromIP(ip string) (GeoInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://ipapi.co/%s/json/", ip))
	if err != nil {
		return GeoInfo{}, err
	}
	defer resp.Body.Close()
	var geo GeoInfo
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return GeoInfo{}, err
	}
	return geo, nil
}
