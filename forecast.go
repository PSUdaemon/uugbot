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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/karlseguin/rcache"
	"github.com/nickvanw/ircx"
	"github.com/sorcix/irc"
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
	caZip   = regexp.MustCompile(`^([ABCEGHJKLMNPRSTVXY]{1}\d{1}[A-Z]{1}) ?\d{1}[A-Z]{1}\d{1}$`)
	usZip   = regexp.MustCompile(`^(\d{5})(-\d{4})?$`)
	causZip = regexp.MustCompile(`(^\d{5}(-\d{4})?$)|(^[ABCEGHJKLMNPRSTVXY]{1}\d{1}[A-Z]{1} ?\d{1}[A-Z]{1}\d{1}$)`)
	cache   = rcache.New(fetcher, time.Hour*24*7)
)

func GetWeather(s ircx.Sender, message *irc.Message) {
	if causZip.MatchString(message.Trailing) {
		var p []string
		var prefix string

		if message.Params[0] == config.General.Name {
			p = []string{message.Prefix.Name}
		} else {
			p = message.Params
			prefix = fmt.Sprint(message.Prefix.Name, ": ")
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

				t := time.Unix(w.Currently.Time, 0).In(l)

				log.Println("Sending weather for", message.Trailing)

				m.Trailing = fmt.Sprintf("%s%s, %s (%.4f, %.4f) %s - %.2f°F (feels like %.2f°F) - %s",
					prefix, z.Places[0].PlaceName, z.Places[0].StateAbbr,
					z.Places[0].Latitude, z.Places[0].Longitude, t,
					w.Currently.Temperature, w.Currently.ApparentTemperature,
					w.Currently.Summary)
				s.Send(m)

				m.Trailing = fmt.Sprintf("%s%d%% Humidity - Wind from %d° at %.2fmph - Visibility %.2fmi - Cloud Cover %d%% - Precipitation Probability %d%%",
					prefix, int(w.Currently.Humidity*100),
					w.Currently.WindBearing, w.Currently.WindSpeed,
					w.Currently.Visibility,
					int(w.Currently.CloudCover*100),
					int(w.Currently.PrecipProbability*100))
				s.Send(m)

				m.Trailing = fmt.Sprintf("%s%s %s %s", prefix, w.Minutely.Summary, w.Hourly.Summary, w.Daily.Summary)
				s.Send(m)
			} else {
				log.Println("No data returned for zip:", message.Trailing)
			}
		}
	}
}
