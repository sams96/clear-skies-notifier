package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var metcheckURL = "https://ws1.metcheck.com/ENGINE/v9_0/json.asp?Fc=As"
var message = `<a href="https://www.metcheck.com/HOBBIES/astronomy_forecast.asp?Lat={{.Lat}}&Lon={{.Lon}}" target="_blank"><img src="https://www.metcheck.com/TRIGGERS/STICKIES/g24_h_astronomy.ASP?Lat={{.Lat}}&Lon={{.Lon}}&LocationName={{.Lat}}/{{.Lon}}&LocType=&Force=TRUE&U=C+DateSelect" border="0" title="Latest Weather Forecast from www.metcheck.com - Click for full forecast"></a>`

type location struct {
	Lat string
	Lon string
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
	query.Add("lat", loc.Lat)
	query.Add("lon", loc.Lon)

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

func sendEmail(recipiant, senderAddress, password, host, port string, loc location) error {
	t, err := template.New("message").Parse(message)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, loc)
	if err != nil {
		return err
	}

	auth := smtp.PlainAuth("", senderAddress, password, host)

	to := []string{recipiant}
	msg := []byte(
		"To: " + recipiant + "\r\n" +
			"Subject: Looks like clear skies tonight\r\n" +
			"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
			buf.String() + "\r\n",
	)

	err = smtp.SendMail(host+":"+port, auth, senderAddress, to, msg)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var (
		loc           = location{os.Getenv("LATITUDE"), os.Getenv("LONGITUDE")}
		recipiant     = os.Getenv("RECIPIANT")
		senderAddress = os.Getenv("SENDER_ADDRESS")
		password      = os.Getenv("PASSWORD")
		host          = os.Getenv("HOST")
		port          = os.Getenv("PORT")
	)
	isClear, err := checkForecast(loc)
	if err != nil {
		panic(err)
	}

	if isClear {
		err = sendEmail(recipiant, senderAddress, password, host, port, loc)
		if err != nil {
			panic(err)
		}
	}
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
