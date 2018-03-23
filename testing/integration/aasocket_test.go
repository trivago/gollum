// +build integration

package integration

import (
	"net"
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	TestConfigSocket = "test_socket.conf"
)

func TestSocketWriteSocket(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(TestConfigSocket, FileStartIndicator("/tmp/integration.socket"), "-ll=2")
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

	err = cmd.Stop()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.Contains(out, "### test UDP")
	expect.Contains(out, "### test Socket")
	expect.Contains(out, "### test TCP")
	expect.Contains(out, "### test ACK")
}

func TestSocketCleanup(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(TestConfigSocket, DefaultStartIndicator, "-ll=3")
	expect.NoError(err)

	socket, err := net.Dial("unix", "/tmp/integration.socket")
	expect.NoError(err)

	tcp, err := net.Dial("tcp", "127.0.0.1:5881")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	dummyData := make([]byte, 8192)
	_, err = socket.Write(dummyData)
	expect.NotNil(err)

	tcp.Write(dummyData)
	_, err = tcp.Read(dummyData)
	expect.NotNil(err)
}
