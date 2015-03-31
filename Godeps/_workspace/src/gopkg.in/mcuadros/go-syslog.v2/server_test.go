package syslog

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/jeromer/syslogparser"
	. "launchpad.net/gocheck"
)

func Test(t *testing.T) { TestingT(t) }

type ServerSuite struct {
}

var _ = Suite(&ServerSuite{})
var exampleSyslog = "<31>Dec 26 05:08:46 hostname tag[296]: content"

func (s *ServerSuite) TestTailFile(c *C) {
	handler := new(HandlerMock)
	server := NewServer()
	server.SetFormat(RFC3164)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:5141")
	server.ListenTCP("0.0.0.0:5141")

	go func(server *Server) {
		time.Sleep(100 * time.Microsecond)

		serverAddr, _ := net.ResolveUDPAddr("udp", "localhost:5141")
		con, _ := net.DialUDP("udp", nil, serverAddr)
		con.Write([]byte(exampleSyslog))
		time.Sleep(100 * time.Microsecond)

		server.Kill()
	}(server)

	server.Boot()
	server.Wait()

	c.Check(handler.LastLogParts["hostname"], Equals, "hostname")
	c.Check(handler.LastLogParts["tag"], Equals, "tag")
	c.Check(handler.LastLogParts["content"], Equals, "content")
	c.Check(handler.LastMessageLength, Equals, int64(len(exampleSyslog)))
	c.Check(handler.LastError, IsNil)
}

type HandlerMock struct {
	LastLogParts      syslogparser.LogParts
	LastMessageLength int64
	LastError         error
}

func (s *HandlerMock) Handle(logParts syslogparser.LogParts, msgLen int64, err error) {
	s.LastLogParts = logParts
	s.LastMessageLength = msgLen
	s.LastError = err
}

type ConnMock struct {
	ReadData       []byte
	ReturnTimeout  bool
	isClosed       bool
	isReadDeadline bool
}

func (c *ConnMock) Read(b []byte) (n int, err error) {
	if c.ReturnTimeout {
		return 0, net.UnknownNetworkError("i/o timeout")
	}
	if c.ReadData != nil {
		l := copy(b, c.ReadData)
		c.ReadData = nil
		return l, nil
	}
	return 0, io.EOF
}

func (c *ConnMock) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (c *ConnMock) Close() error {
	c.isClosed = true
	return nil
}

func (c *ConnMock) LocalAddr() net.Addr {
	return nil
}

func (c *ConnMock) RemoteAddr() net.Addr {
	return nil
}

func (c *ConnMock) SetDeadline(t time.Time) error {
	return nil
}

func (c *ConnMock) SetReadDeadline(t time.Time) error {
	c.isReadDeadline = true
	return nil
}

func (c *ConnMock) SetWriteDeadline(t time.Time) error {
	return nil
}

func (s *ServerSuite) TestConnectionClose(c *C) {
	for _, closeConnection := range []bool{true, false} {
		handler := new(HandlerMock)
		server := NewServer()
		server.SetFormat(RFC3164)
		server.SetHandler(handler)
		con := ConnMock{ReadData: []byte(exampleSyslog)}
		server.goScanConnection(&con, closeConnection)
		server.Wait()
		c.Check(con.isClosed, Equals, closeConnection)
	}
}

func (s *ServerSuite) TestTcpTimeout(c *C) {
	handler := new(HandlerMock)
	server := NewServer()
	server.SetFormat(RFC3164)
	server.SetHandler(handler)
	server.SetTimeout(10)
	con := ConnMock{ReadData: []byte(exampleSyslog), ReturnTimeout: true}
	c.Check(con.isReadDeadline, Equals, false)
	server.goScanConnection(&con, true)
	server.Wait()
	c.Check(con.isReadDeadline, Equals, true)
	c.Check(handler.LastLogParts, IsNil)
	c.Check(handler.LastMessageLength, Equals, int64(0))
	c.Check(handler.LastError, IsNil)
}
