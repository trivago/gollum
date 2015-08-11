package shared

import (
	"fmt"
	"log"
	"net"
	"time"
)

// MetricServer contains state information about the metric server process
type MetricServer struct {
	running bool
	listen  net.Listener
}

// NewMetricServer creates a new server state for a metric server
func NewMetricServer() *MetricServer {
	return &MetricServer{
		running: false,
	}
}

func (server *MetricServer) handleMetricRequest(conn net.Conn) {
	defer conn.Close()

	data, err := Metric.Dump()
	if err != nil {
		conn.Write([]byte(err.Error()))
	} else {
		conn.Write(data)
	}
	conn.Write([]byte{'\n'})
	conn.Close()
}

// Start causes a metric server to listen for a specific port.
// If this port is accessed a JSON containing all metrics will be returned and
// the connection is closed.
func (server *MetricServer) Start(port int) {
	if server.running {
		return
	}

	var err error
	server.listen, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Print("Metrics: ", err)
		time.AfterFunc(time.Second*5, func() { server.Start(port) })
		return
	}

	server.running = true
	for server.running {
		client, err := server.listen.Accept()
		if err != nil {
			if server.running {
				log.Print("Metrics: ", err)
			}
			return // ### break ###
		}

		go server.handleMetricRequest(client)
	}
}

// Stop notifies the metric server to halt.
func (server *MetricServer) Stop() {
	server.running = false
	if server.listen != nil {
		if err := server.listen.Close(); err != nil {
			log.Print("Metrics: ", err)
		}
	}
}
