// Copyright 2015-2017 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package producer

import (
	"bytes"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// influxDBWriter08 implements the io.Writer interface for InfluxDB 0.9 connections
type influxDBWriter08 struct {
	client           http.Client
	connectionUp     bool
	host             string
	username         string
	password         string
	writeURL         string
	testURL          string
	databaseTemplate string
	timeBasedDBName  bool
	Control          func() chan<- core.PluginControl
	buffer           shared.ByteStream
}

// Configure sets the database connection values
func (writer *influxDBWriter08) configure(conf core.PluginConfig, prod *InfluxDB) error {
	writer.host = conf.GetString("Host", "localhost:8086")
	writer.username = conf.GetString("User", "")
	writer.password = conf.GetString("Password", "")
	writer.databaseTemplate = conf.GetString("Database", "default")
	writer.buffer = shared.NewByteStream(4096)
	writer.connectionUp = false
	writer.timeBasedDBName = conf.GetBool("TimeBasedName", true)
	writer.Control = prod.Control

	writer.writeURL = fmt.Sprintf("http://%s/db/%%s/series?time_precision=ms", writer.host)
	writer.testURL = fmt.Sprintf("http://%s/db", writer.host)

	if writer.username != "" {
		credentials := fmt.Sprintf("u=%s&p=%s", url.QueryEscape(writer.username), url.QueryEscape(writer.password))
		writer.writeURL += "&" + credentials
		writer.testURL += "?" + credentials
	}

	prod.SetCheckFuseCallback(writer.isConnectionUp)
	return nil
}

func (writer *influxDBWriter08) isConnectionUp() bool {
	if writer.connectionUp {
		return true // ### return, connection not reported to be down ###
	}

	if response, err := http.Get(writer.testURL); err == nil && response != nil {
		defer response.Body.Close()
		switch response.Status[:3] {
		case "200":
			writer.connectionUp = true
			Log.Note.Print("Connected to " + writer.host)
		}
	}

	writer.Control() <- core.PluginControlFuseActive
	return writer.connectionUp
}

func (writer *influxDBWriter08) createDatabase(database string) error {
	url := fmt.Sprintf("http://%s/db?u=%s&p=%s", writer.host, url.QueryEscape(writer.username), url.QueryEscape(writer.password))
	body := fmt.Sprintf(`{"name": "%s"}`, database)

	response, err := writer.client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}

	defer response.Body.Close()
	switch response.Status[:3] {
	case "201":
		Log.Note.Printf("Created database %s", database)
		return nil

	default:
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("Could not create database %s with status code \"%s\" and error \"%s\"", database, response.Status[:3], body)
	}
}

func (writer *influxDBWriter08) post() (int, error) {
	databaseName := writer.databaseTemplate
	if writer.timeBasedDBName {
		databaseName = time.Now().Format(databaseName)
	}

	writeURL := fmt.Sprintf(writer.writeURL, url.QueryEscape(databaseName))
	response, err := writer.client.Post(writeURL, "application/json", &writer.buffer)
	if err != nil {
		writer.connectionUp = false
		writer.Control() <- core.PluginControlFuseBurn
		return 0, err // ### return, failed to connect ###
	}

	defer response.Body.Close()

	// Check status codes
	switch response.Status[:3] {
	case "200":
		return writer.buffer.Len(), nil // ### return, OK ###

	case "400":
		body, _ := ioutil.ReadAll(response.Body)
		// 400 Bad Request: Database foobar doesn't exist
		if strings.Contains(string(body), databaseName) {
			err := writer.createDatabase(databaseName)
			if err != nil {
				return 0, err // ### return, failed to create database ###
			}
			writer.buffer.ResetRead()
			return writer.post() // ### return, retry ###
		}
		fallthrough

	default:
		body, _ := ioutil.ReadAll(response.Body)
		writer.connectionUp = false
		return 0, fmt.Errorf("%s returned %s: %s", writeURL, response.Status, string(body))
	}
}

func (writer *influxDBWriter08) Write(data []byte) (int, error) {
	data = bytes.TrimRight(data, " \t,") // remove the last comma

	writer.buffer.Reset()
	writer.buffer.WriteByte('[')
	writer.buffer.Write(data)
	writer.buffer.WriteByte(']')

	return writer.post()
}
