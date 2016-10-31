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
	"time"

	"gopkg.in/sorcix/irc.v1"
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
		Nick     string
		Server   string
		Channels []struct {
			Name string
			Pass string
		}
	}
	Forecast struct {
		Key string
	}
}

func main() {
	var server_conn *irc.Conn

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

	log.Println(config)

	for {
		var m *irc.Message
		var e error

		for server_conn == nil {
			server_conn, err = irc.Dial(config.General.Server)
			if e != nil {
				log.Println(e.Error())
				server_conn = nil
				time.Sleep(10 * time.Second)
			}
		}

		log.Println("Connected to Server")
		log.Println("Sending Nick")
		server_conn.Encode(&irc.Message{
			Command: irc.NICK,
			Params:  []string{config.General.Nick},
		})

		log.Println("Sending User")
		server_conn.Encode(&irc.Message{
			Command:  irc.USER,
			Params:   []string{config.General.Nick, "0", "*"},
			Trailing: config.General.Nick,
		})

		for {
			m, e = server_conn.Decode()
			if e == nil {
				switch m.Command {
				case irc.PING:
					log.Println("Received PING")
					server_conn.Encode(&irc.Message{
						Command:  irc.PONG,
						Params:   m.Params,
						Trailing: m.Trailing,
					})
				case irc.RPL_WELCOME:
					log.Println("Received Welcome")
					for _, irc_chan := range config.General.Channels {
						log.Println("Joining ", irc_chan.Name)
						server_conn.Encode(&irc.Message{
							Command: irc.JOIN,
							Params:  []string{irc_chan.Name, irc_chan.Pass},
						})
					}
				case irc.PRIVMSG:
					log.Println("Received PRIVMSG")
					go GetWeather(server_conn.Encoder, m)
					go GetTitle(server_conn.Encoder, m)
				default:
					log.Println(m)
				}
			} else {
				log.Println(e.Error())
				e = server_conn.Close()
				if e != nil {
					log.Println(e.Error())
				}
				server_conn = nil
				break
			}
		}
	}
	log.Println("Exiting..")
}
