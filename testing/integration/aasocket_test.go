// +build integration

package integration

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	TestConfigSocket = "test_socket.conf"
)

func TestSocketWriteSocket(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(TestConfigSocket, "-ll=2")
	expect.NoError(err)

	udp, err := net.Dial("udp", "127.0.0.1:5880")
	expect.NoError(err)

	socket, err := net.Dial("unix", "/tmp/integration.socket")
	expect.NoError(err)

	tcp, err := net.Dial("tcp", "127.0.0.1:5881")
	expect.NoError(err)

	tcpAck, err := net.Dial("tcp", "127.0.0.1:5882")
	expect.NoError(err)

	udp.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = udp.Write([]byte("### test UDP\n"))
	udp.Close()
	expect.NoError(err)

	socket.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = socket.Write([]byte("### test Socket\n"))
	socket.Close()
	expect.NoError(err)

	tcp.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = tcp.Write([]byte("### test TCP\n"))
	tcp.Close()
	expect.NoError(err)

	tcpAck.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = tcpAck.Write([]byte("### test ACK\n"))
	expect.NoError(err)

	buffer := make([]byte, 8)
	tcpAck.SetReadDeadline(time.Now().Add(time.Second))
	n, err := tcpAck.Read(buffer)

	tcpAck.Close()
	expect.NoError(err)
	expect.Equal([]byte("ACK"), buffer[:n])

	out := StopGollum(cmd)

	buffer = make([]byte, 4096)
	n, err = out.Read(buffer)
	expect.NoError(err)

	text := string(buffer[:n])
	expect.Contains(text, "### test UDP")
	expect.Contains(text, "### test Socket")
	expect.Contains(text, "### test TCP")
	expect.Contains(text, "### test ACK")
}

func TestSocketCleanup(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(TestConfigSocket, "-ll=3")
	expect.NoError(err)

	socket, err := net.Dial("unix", "/tmp/integration.socket")
	expect.NoError(err)

	tcp, err := net.Dial("tcp", "127.0.0.1:5881")
	expect.NoError(err)

	var out *bytes.Buffer
	expect.NonBlocking(10*time.Second, func() {
		out = StopGollum(cmd)
	})

	buffer := make([]byte, 8192)
	_, err = out.Read(buffer)
	expect.NoError(err)

	_, err = socket.Write(buffer)
	expect.NotNil(err)

	tcp.Write(buffer)
	_, err = tcp.Read(buffer)
	expect.NotNil(err)
}
