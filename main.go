package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var metcheckURL = "https://ws1.metcheck.com/ENGINE/v9_0/json.asp?Fc=As"

type location struct {
	lat float64
	lon float64
}

type metcheckResponse struct {
	MetcheckData struct {
		ForecastLocation struct {
			Forecast []forecast `json:"forecast"`
		} `json:"forecastLocation"`
	} `json:"metcheckData"`
}
type forecast struct {
	SeeingIndex string   `json:"seeingIndex"`
	Time        *utcTime `json:"utcTime"`
}

func checkForecast(loc location) (bool, error) {
	URL, err := url.Parse(metcheckURL)
	if err != nil {
		return false, err
	}

	query := URL.Query()
	query.Add("lat", strconv.FormatFloat(loc.lat, 'f', 4, 64))
	query.Add("lon", strconv.FormatFloat(loc.lon, 'f', 4, 64))

	resp, err := http.Get(URL.String())
	if err != nil {
		return false, err
	}

	var body metcheckResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return false, err
	}

	for _, forecast := range body.MetcheckData.ForecastLocation.Forecast {
		t := time.Time(*forecast.Time)
		n := time.Now()
		endOfNight := time.Date(n.Year(), n.Month(), n.Day(), 4, 0, 0, 0, time.Local)
		if t.After(endOfNight) {
			continue
		}

		index, err := strconv.ParseInt(forecast.SeeingIndex, 10, 16)
		if err != nil {
			return false, err
		}

		if index >= 7 {
			return true, nil
		}
	}

	return false, nil
}

func main() {
	isClear, err := checkForecast(location{47.268, 8.4108})
	if err != nil {
		panic(err)
	}

	fmt.Println(isClear)
}

type utcTime time.Time

func (t *utcTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	time, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.UTC)
	if err != nil {
		return err
	}

	*t = utcTime(time)
	return nil
}

func (t utcTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t))
}
