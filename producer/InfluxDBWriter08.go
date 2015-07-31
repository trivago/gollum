// Copyright 2015 trivago GmbH
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
	"regexp"
	"strconv"
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
	buffer           shared.ByteStream
	db404Pattern     *regexp.Regexp
}

// Configure sets the database connection values
func (writer *influxDBWriter08) configure(conf core.PluginConfig) error {
	writer.host = conf.GetString("Host", "localhost:8086")
	writer.username = conf.GetString("User", "")
	writer.password = conf.GetString("Password", "")
	writer.databaseTemplate = conf.GetString("Database", "default")
	writer.db404Pattern, _ = regexp.Compile("Database (.*?) doesn't exist")
	writer.buffer = shared.NewByteStream(4096)
	writer.connectionUp = false

	writer.writeURL = fmt.Sprintf("http://%s/db/%%s/series?time_precision=s", writer.host) //   curl -X POST -d ''  'http://10.1.3.220:8086/db/annotations/series?u=root&p=root&time_precision=s'
	writer.testURL = fmt.Sprintf("http://%s/db/", writer.host)
	if writer.username != "" {
		credentials := fmt.Sprintf("&u=%s&p=%s", writer.username, writer.password)
		writer.writeURL += credentials
		writer.testURL += credentials
	}
	writer.testURL = url.QueryEscape(writer.testURL)

	return nil
}

func (writer *influxDBWriter08) isConnectionUp() bool {
	if writer.connectionUp {
		return true // ### return, connection not reported to be down ###
	}
	if response, err := http.Get(writer.testURL); err != nil {
		if status, err := strconv.Atoi(response.Status[:3]); err != nil && status == 200 {
			writer.connectionUp = true
		}
	}
	return writer.connectionUp
}

func (writer *influxDBWriter08) createDatabase(database string) error {
	url := url.QueryEscape(fmt.Sprintf("http://%s/db?u=%s&p=%s", writer.host, writer.username, writer.password))
	body := fmt.Sprintf(`{"name": "%s"}`, database)
	Log.Debug.Print(url)
	Log.Debug.Print(body)

	response, err := writer.client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}

	defer response.Body.Close()
	status, _ := strconv.Atoi(response.Status[:3])

	switch status {
	case 201: // 201 = created
		return nil

	default:
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("Could not create database %s with status code %d and error %s", database, status, body)
	}
}

func (writer *influxDBWriter08) post() (int, error) {
	database := time.Now().Format(writer.databaseTemplate) // Allow timestamping the database with the current time
	writeURL := url.QueryEscape(fmt.Sprintf(writer.writeURL, database))

	response, err := writer.client.Post(writeURL, "application/json", &writer.buffer)
	if err != nil {
		writer.connectionUp = false
		return 0, err // ### return, failed to connect ###
	}

	defer response.Body.Close()

	// Check status codes
	status, _ := strconv.Atoi(response.Status[:3])
	switch status {
	case 200:
		return writer.buffer.Len(), nil // ### return, OK ###

	case 400:
		body, _ := ioutil.ReadAll(response.Body)
		// 400 Bad Request: Database foobar doesn't exist
		if matches := writer.db404Pattern.FindStringSubmatch(string(body)); matches != nil {
			databaseName := matches[1]
			if databaseName != "" {
				Log.Debug.Printf("Creating database %s", databaseName)
				err := writer.createDatabase(databaseName)
				if err != nil {
					return 0, err // ### return, failed to create database ###
				}
				return writer.post() // ### return, retry ###
			}
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
