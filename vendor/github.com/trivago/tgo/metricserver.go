package tgo

import (
	"log"
	"net"
	"time"
)

// MetricServer contains state information about the metric server process
type MetricServer struct {
	metrics *Metrics
	running bool
	listen  net.Listener
	updates *time.Ticker
}

// NewMetricServer creates a new server state for a metric server based on
// the global Metric variable.
func NewMetricServer() *MetricServer {
	return &MetricServer{
		metrics: Metric,
		running: false,
		updates: time.NewTicker(time.Second),
	}
}

// NewMetricServerFor creates a new server state for a metric server based
// on a custom Metrics variable
func NewMetricServerFor(m *Metrics) *MetricServer {
	return &MetricServer{
		metrics: m,
		running: false,
		updates: time.NewTicker(time.Second),
	}
}

func (server *MetricServer) handleMetricRequest(conn net.Conn) {
	defer conn.Close()

	data, err := server.metrics.Dump()
	if err != nil {
		conn.Write([]byte(err.Error()))
	} else {
		conn.Write(data)
	}
	conn.Write([]byte{'\n'})
	conn.Close()
}

func (server *MetricServer) sysUpdate() {
	for server.running {
		_, running := <-server.updates.C
		if !running {
			return // ### return, timer has been stopped ###
		}

		server.metrics.UpdateSystemMetrics()
	}
}

// Start causes a metric server to listen for a specific address and port.
// If this address/port is accessed a JSON containing all metrics will be
// returned and the connection is closed.
// You can use the standard go notation for addresses like ":80".
func (server *MetricServer) Start(address string) {
	if server.running {
		return
	}

	var err error
	server.listen, err = net.Listen("tcp", address)
	if err != nil {
		log.Print("Metrics: ", err)
		time.AfterFunc(time.Second*5, func() { server.Start(address) })
		return
	}

	server.running = true
	go server.sysUpdate()

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
	server.updates.Stop()
	if server.listen != nil {
		if err := server.listen.Close(); err != nil {
			log.Print("Metrics: ", err)
		}
	}
}
