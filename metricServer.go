package main

import (
	"fmt"
	"github.com/trivago/gollum/log"
	"net"
)

func startMetricServer(port int) {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		Log.Error.Print("Metrics: ", err)
		return
	}

	client, err := listen.Accept()
	if err != nil {
		Log.Error.Print("Metrics: ", err)
		return // ### break ###
	}

	go handleMetricRequest(client)
}

func handleMetricRequest(conn net.Conn) {
	defer conn.Close()

	data, err := Log.Metric.Dump()
	if err != nil {
		conn.Write([]byte(err.Error()))
	} else {
		conn.Write(data)
	}

	conn.Close()
}
