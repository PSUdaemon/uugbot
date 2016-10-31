/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"text/template"
	"time"

	"github.com/karlseguin/rcache"
	"gopkg.in/sorcix/irc.v1"
)

type ZipInfo struct {
	Country     string `json:"country"`
	CountryAbbr string `json:"country abbreviation"`
	PostCode    string `json:"post code"`
	Places      []struct {
		Latitude  float64 `json:"latitude,string"`
		Longitude float64 `json:"longitude,string"`
		PlaceName string  `json:"place name"`
		State     string  `json:"state"`
		StateAbbr string  `json:"state abbreviation"`
	} `json:"places"`
}

type WeatherReport struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Offset    int64   `json:"offset"`
	Timezone  string  `json:"timezone"`
	Currently struct {
		ApparentTemperature  float64 `json:"apparentTemperature"`
		CloudCover           float64 `json:"cloudCover"`
		DewPoint             float64 `json:"dewPoint"`
		Humidity             float64 `json:"humidity"`
		Icon                 string  `json:"icon"`
		NearestStormBearing  int     `json:"nearestStormBearing"`
		NearestStormDistance int64   `json:"nearestStormDistance"`
		Ozone                float64 `json:"ozone"`
		PrecipIntensity      float64 `json:"precipIntensity"`
		PrecipProbability    float64 `json:"precipProbability"`
		Pressure             float64 `json:"pressure"`
		Summary              string  `json:"summary"`
		Temperature          float64 `json:"temperature"`
		Time                 int64   `json:"time"`
		Visibility           float64 `json:"visibility"`
		WindBearing          int     `json:"windBearing"`
		WindSpeed            float64 `json:"windSpeed"`
	} `json:"currently"`
	Minutely struct {
		Data []struct {
			PrecipIntensity   float64 `json:"precipIntensity"`
			PrecipProbability float64 `json:"precipProbability"`
			Time              int64   `json:"time"`
		} `json:"data"`
		Icon    string `json:"icon"`
		Summary string `json:"summary"`
	} `json:"minutely"`
	Hourly struct {
		Data []struct {
			ApparentTemperature float64 `json:"apparentTemperature"`
			CloudCover          float64 `json:"cloudCover"`
			DewPoint            float64 `json:"dewPoint"`
			Humidity            float64 `json:"humidity"`
			Icon                string  `json:"icon"`
			Ozone               float64 `json:"ozone"`
			PrecipIntensity     float64 `json:"precipIntensity"`
			PrecipProbability   float64 `json:"precipProbability"`
			Pressure            float64 `json:"pressure"`
			Summary             string  `json:"summary"`
			Temperature         float64 `json:"temperature"`
			Time                int64   `json:"time"`
			Visibility          float64 `json:"visibility"`
			WindBearing         int     `json:"windBearing"`
			WindSpeed           float64 `json:"windSpeed"`
		} `json:"data"`
		Icon    string `json:"icon"`
		Summary string `json:"summary"`
	} `json:"hourly"`
	Daily struct {
		Data []struct {
			ApparentTemperatureMax     float64 `json:"apparentTemperatureMax"`
			ApparentTemperatureMaxTime int64   `json:"apparentTemperatureMaxTime"`
			ApparentTemperatureMin     float64 `json:"apparentTemperatureMin"`
			ApparentTemperatureMinTime int64   `json:"apparentTemperatureMinTime"`
			CloudCover                 float64 `json:"cloudCover"`
			DewPoint                   float64 `json:"dewPoint"`
			Humidity                   float64 `json:"humidity"`
			Icon                       string  `json:"icon"`
			MoonPhase                  float64 `json:"moonPhase"`
			Ozone                      float64 `json:"ozone"`
			PrecipIntensity            float64 `json:"precipIntensity"`
			PrecipIntensityMax         float64 `json:"precipIntensityMax"`
			PrecipIntensityMaxTime     int64   `json:"precipIntensityMaxTime"`
			PrecipProbability          float64 `json:"precipProbability"`
			PrecipType                 string  `json:"precipType"`
			Pressure                   float64 `json:"pressure"`
			Summary                    string  `json:"summary"`
			SunriseTime                int64   `json:"sunriseTime"`
			SunsetTime                 int64   `json:"sunsetTime"`
			TemperatureMax             float64 `json:"temperatureMax"`
			TemperatureMaxTime         int64   `json:"temperatureMaxTime"`
			TemperatureMin             float64 `json:"temperatureMin"`
			TemperatureMinTime         int64   `json:"temperatureMinTime"`
			Time                       int64   `json:"time"`
			Visibility                 float64 `json:"visibility"`
			WindBearing                int     `json:"windBearing"`
			WindSpeed                  float64 `json:"windSpeed"`
		} `json:"data"`
		Icon    string `json:"icon"`
		Summary string `json:"summary"`
	} `json:"daily"`
	Flags struct {
		DarkskyStations []string `json:"darksky-stations"`
		IsdStations     []string `json:"isd-stations"`
		MadisStations   []string `json:"madis-stations"`
		Sources         []string `json:"sources"`
		Units           string   `json:"units"`
	} `json:"flags"`
}

type TemplateForecast struct {
	Nick             string
	CurrentCond      string
	Temp             float64
	ApparentTemp     float64
	TempUnits        string
	WindDirection    int
	WindSpeed        float64
	WindUnits        string
	Visibility       float64
	VisibilityUnits  string
	Pressure         float64
	PressureUnits    string
	RelativeHumidity int
	CloudCover       int
	PrecipProb       int
	Sunrise          string
	Sunset           string
	LocationName     string
	LocationState    string
}

func fetcher(key string) interface{} {
	var z ZipInfo
	var err error
	var resp *http.Response

	if usZip.MatchString(key) {
		zip := usZip.FindStringSubmatch(key)[1]
		log.Println("Looking up coordinates for US zip:", zip)
		resp, err = http.Get(fmt.Sprintf("http://api.zippopotam.us/us/%s", zip))
	} else if caZip.MatchString(key) {
		zip := caZip.FindStringSubmatch(key)[1]
		log.Println("Looking up coordinates for CA postal code:", zip)
		resp, err = http.Get(fmt.Sprintf("http://api.zippopotam.us/ca/%s", zip))
	}

	if err != nil {
		log.Printf("Lookup failed for zip: %s (%s)\n", key, err)
		return nil
	}

	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&z)

	if err != nil {
		log.Printf("Unable to parse result for zip: %s (%s)\n", key, err)
		return nil
	}

	return &z
}

var (
	caZip       = regexp.MustCompile(`^([ABCEGHJKLMNPRSTVXY]{1}\d{1}[A-Z]{1}) ?\d{1}[A-Z]{1}\d{1}$`)
	usZip       = regexp.MustCompile(`^(\d{5})(-\d{4})?$`)
	causZip     = regexp.MustCompile(`(^\d{5}(-\d{4})?$)|(^[ABCEGHJKLMNPRSTVXY]{1}\d{1}[A-Z]{1} ?\d{1}[A-Z]{1}\d{1}$)`)
	cache       = rcache.New(fetcher, time.Hour*24*7)
	tmpl        = template.Must(template.ParseGlob("tmpl/*"))
	icon_lookup = map[string]string{
		"clear-day":           "Sunny",
		"clear-night":         "Sunny",
		"rain":                "LightRain",
		"snow":                "LightSnow",
		"sleet":               "LightSleet",
		"wind":                "PartlyCloudy",
		"fog":                 "Fog",
		"cloudy":              "Cloudy",
		"partly-cloudy-day":   "PartlyCloudy",
		"partly-cloudy-night": "PartlyCloudy",
		"hail":                "Unknown",
		"thunderstorm":        "ThunderyShowers",
		"tornado":             "Unknown",
	}
)

func GetWeather(e irc.Encoder, message *irc.Message) {
	if causZip.MatchString(message.Trailing) {
		var p []string
		var prefix string

		if message.Params[0] == config.General.Nick {
			p = []string{message.Prefix.Name}
		} else {
			p = message.Params
			prefix = message.Prefix.Name
		}

		m := &irc.Message{
			Command: irc.PRIVMSG,
			Params:  p,
		}

		zl := cache.Get(message.Trailing)

		if zl != nil {
			z := zl.(*ZipInfo)
			if z.Places != nil {
				resp, err := http.Get(fmt.Sprintf("https://api.forecast.io/forecast/%s/%.4f,%.4f?exclude=flags",
					config.Forecast.Key, z.Places[0].Latitude, z.Places[0].Longitude))
				if err != nil {
					// handle error
					return
				}
				defer resp.Body.Close()

				dec := json.NewDecoder(resp.Body)

				var w WeatherReport
				err = dec.Decode(&w)

				l, _ := time.LoadLocation(w.Timezone)

				log.Println("Sending weather for", message.Trailing)

				forecast := &TemplateForecast{
					Nick:             prefix,
					CurrentCond:      w.Currently.Summary,
					Temp:             w.Currently.Temperature,
					ApparentTemp:     w.Currently.ApparentTemperature,
					TempUnits:        "F",
					WindDirection:    w.Currently.WindBearing,
					WindSpeed:        w.Currently.WindSpeed,
					WindUnits:        "mph",
					Visibility:       w.Currently.Visibility,
					VisibilityUnits:  "mi",
					Pressure:         w.Currently.Pressure,
					PressureUnits:    "mbar",
					RelativeHumidity: int(w.Currently.Humidity * 100),
					CloudCover:       int(w.Currently.CloudCover * 100),
					PrecipProb:       int(w.Currently.PrecipProbability * 100),
					Sunrise:          time.Unix(w.Daily.Data[0].SunriseTime, 0).In(l).Format(time.Kitchen),
					Sunset:           time.Unix(w.Daily.Data[0].SunsetTime, 0).In(l).Format(time.Kitchen),
					LocationName:     z.Places[0].PlaceName,
					LocationState:    z.Places[0].StateAbbr,
				}

				var buffer bytes.Buffer

				tmpl.ExecuteTemplate(&buffer, icon_lookup[w.Currently.Icon], forecast)
				scanner := bufio.NewScanner(&buffer)

				for scanner.Scan() {
					m.Trailing = scanner.Text()
					e.Encode(m)
				}
			} else {
				log.Println("No data returned for zip:", message.Trailing)
			}
		}
	}
}
