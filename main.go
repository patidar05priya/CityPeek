package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var funFacts = map[string]string{
	"tokyo": "Tokyo was once called Edo",
}

type Response struct {
	City    string `json:"city"`
	Time    string `json:"time"`
	Weather string `json:"weather"`
	Fact    string `json:"fact"`
}

func main() {
	log.Printf("OPENWEATHER_API_KEY: %s", os.Getenv("OPENWEATHER_API_KEY"))
	log.Printf("TIMEZONE_API: %s", os.Getenv("TIMEZONE_API"))
	http.HandleFunc("/cityPeek", handler)
	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {

	city := strings.ToLower(r.URL.Query().Get("city"))
	if city == "" {
		http.Error(w, "city is required", http.StatusBadRequest)
		return
	}

	weather, lat, lon := getWeather(city)
	localTime := getTimeFromLatLon(lat, lon)
	fact := funFacts[city]

	resp := Response{
		City:    strings.Title(city),
		Time:    localTime,
		Weather: weather,
		Fact:    fact,
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getTimeFromLatLon(lat, lon float64) string {
	timeZoneAPIKey := os.Getenv("TIMEZONE_API")
	apiURL := fmt.Sprintf(
		"https://api.timezonedb.com/v2.1/get-time-zone?key=%s&format=json&by=position&lat=%f&lng=%f",
		timeZoneAPIKey, lat, lon)

	resp, err := http.Get(apiURL)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println(err)
		fmt.Println("Get ", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println(err)
		return "Unavailable"
	}

	if t, ok := data["formatted"].(string); ok {
		return t
	}
	return "Unavailable"
}

func getWeather(city string) (string, float64, float64) {
	base := "https://api.openweathermap.org/data/2.5/weather"
	q := url.QueryEscape(city)
	weatherAPIKey := os.Getenv("OPENWEATHER_API_KEY")

	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", base, q, weatherAPIKey)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println("Get ", err)
		return "Unavailable", 0, 0
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println(err)
		return "Unavailable", 0, 0
	}

	// Get coordinates
	coordData, ok := result["coord"].(map[string]interface{})
	if !ok {
		log.Println("Missing or invalid coord in weather API response")
		return "Unavailable", 0, 0
	}
	lat, latOk := coordData["lat"].(float64)
	lon, lonOk := coordData["lon"].(float64)
	if !latOk || !lonOk {
		log.Println("Missing lat/lon values")
		return "Unavailable", 0, 0
	}

	// Weather details
	main := result["main"].(map[string]interface{})
	weatherArr := result["weather"].([]interface{})
	desc := weatherArr[0].(map[string]interface{})["description"].(string)

	temp := fmt.Sprintf("%.1f°C", main["temp"].(float64))
	humidity := fmt.Sprintf("%v%%", main["humidity"])

	return fmt.Sprintf("%s, %s, Humidity: %s", temp, desc, humidity), lat, lon
}
