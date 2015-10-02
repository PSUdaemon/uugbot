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
	"os"

	"github.com/nickvanw/ircx"
	"github.com/sorcix/irc"
	"gopkg.in/yaml.v2"
)

var (
	configfile = flag.String("config", "config.yaml", "Path to config file")
	config     Config
)

func init() {
	flag.Parse()
}

type Config struct {
	General struct {
		Name     string
		Server   string
		Channels []struct {
			Name string
			Pass string
		}
	}
	Forecast struct{ Key string }
}

func RegisterHandlers(bot *ircx.Bot) {
	bot.HandleFunc(irc.RPL_WELCOME, RegisterConnect)
	bot.HandleFunc(irc.PING, PingHandler)
	bot.HandleFunc(irc.PRIVMSG, PrivMsgHandler)
}

func main() {
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
	bot.HandleLoop()
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

func PrivMsgHandler(s ircx.Sender, message *irc.Message) {
	go GetWeather(s, message)
	go GetTitle(s, message)
}
