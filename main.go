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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/nickvanw/ircx"
	"github.com/sorcix/irc"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"gopkg.in/yaml.v2"
)

var (
	configfile = flag.String("config", "config.yaml", "Path to config file")
	config     Config
)

func init() {
	flag.Parse()
}

type Channel struct {
	Name string
	Pass string
}

type BotConfig struct {
	Name     string
	Server   string
	Channels []Channel
}

type Config struct {
	General  BotConfig
	Forecast struct{ Key string }
}

func RegisterHandlers(bot *ircx.Bot) {
	bot.AddCallback(irc.RPL_WELCOME, ircx.Callback{Handler: ircx.HandlerFunc(RegisterConnect)})
	bot.AddCallback(irc.PING, ircx.Callback{Handler: ircx.HandlerFunc(PingHandler)})
	bot.AddCallback(irc.PRIVMSG, ircx.Callback{Handler: ircx.HandlerFunc(PrivMsgHandler)})
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	b, err := ioutil.ReadFile(*configfile)

	if err != nil {
		fmt.Printf("Can not read file (%s)\n", *configfile)
		os.Exit(1)
	}

	err = yaml.Unmarshal(b, &config)

	if err != nil {
		fmt.Printf("Config (%s) is invalid\n", *configfile)
		os.Exit(1)
	}

	bot := ircx.Classic(config.General.Server, config.General.Name)
	if err := bot.Connect(); err != nil {
		log.Panicln("Unable to dial IRC Server ", err)
	}

	RegisterHandlers(bot)
	bot.CallbackLoop()
	log.Println("Exiting..")
}

func PingHandler(s ircx.Sender, m *irc.Message) {
	s.Send(&irc.Message{
		Command:  irc.PONG,
		Params:   m.Params,
		Trailing: m.Trailing,
	})
}

func RegisterConnect(s ircx.Sender, m *irc.Message) {
	for _, irc_chan := range config.General.Channels {
		s.Send(&irc.Message{
			Command: irc.JOIN,
			Params:  []string{irc_chan.Name, irc_chan.Pass},
		})
	}
}

func getURL(s ircx.Sender, message *irc.Message, word string) {
	if url, err := url.Parse(word); err == nil && strings.HasPrefix(url.Scheme, "http") {
		resp, err := http.Head(url.String())
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.Contains(resp.Header.Get("content-type"), "text/html") && resp.ContentLength < 1024*1024 {
				resp2, err2 := http.Get(url.String())
				if err2 == nil {
					defer resp2.Body.Close()
					if resp.StatusCode == 200 && strings.Contains(resp.Header.Get("content-type"), "text/html") {
						z := html.NewTokenizer(resp2.Body)
						for {
							tt := z.Next()
							if tt == html.ErrorToken {
								return
							}
							if tt == html.StartTagToken && z.Token().DataAtom == atom.Title {
								tt = z.Next()
								if tt == html.TextToken {
									p := message.Params
									if p[0] == config.General.Name {
										p = []string{message.Prefix.Name}
									}

									title := fmt.Sprintf("Title: %s", z.Token().Data)

									log.Println("Sending HTML", title)
									s.Send(&irc.Message{
										Command:  irc.PRIVMSG,
										Params:   p,
										Trailing: title,
									})
								}
								return
							}

						}
					}
				}
			}
		}
	}
}

func GetTitle(s ircx.Sender, message *irc.Message) {
	words := strings.Split(message.Trailing, " ")
	for _, word := range words {
		go getURL(s, message, word)
	}
}

func PrivMsgHandler(s ircx.Sender, message *irc.Message) {
	go GetWeather(s, message)
	go GetTitle(s, message)
}
