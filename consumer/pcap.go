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

package consumer

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/miekg/pcap"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type stream struct {
	Key     string
	Timer   *time.Timer
	Packets streamPackets
	Buf     bytes.Buffer
}

type streamPackets []*pcap.Packet

const streamTimeOut = 3 * time.Second

func (a streamPackets) Len() int           { return len(a) }
func (a streamPackets) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a streamPackets) Less(i, j int) bool { return seqNum(a[i]) < seqNum(a[j]) }

func getKey(pkt *pcap.Packet) (string, error) {
	if len(pkt.Headers) == 2 {
		ipHeader := pkt.Headers[0].(*pcap.Iphdr)
		tcpHeader := pkt.Headers[1].(*pcap.Tcphdr)
		key := fmt.Sprintf("%s:%d-%s:%d", ipHeader.SrcAddr(), tcpHeader.SrcPort, ipHeader.DestAddr(), tcpHeader.DestPort)
		return key, nil
	}
	return "", fmt.Errorf("Invalid Packet")
}

func readerDone(reader *bufio.Reader) bool {
	_, err := reader.Peek(1)
	return err != nil
}

func newStream(key string, p *Pcap) *stream {
	return &stream{
		Key:   key,
		Timer: time.AfterFunc(streamTimeOut, func() { delete(p.Openstreams, key) }),
	}
}

func (s *stream) reqsFromBuffer(reqBuffer *bytes.Buffer) ([]*bytes.Buffer, error) {
	var reqArray []*bytes.Buffer
	reqRead := bufio.NewReader(reqBuffer)

	for !readerDone(reqRead) {
		req, err := http.ReadRequest(reqRead)
		if err != nil {
			Log.Error.Print("Pcap: ", err)
			return nil, err
		}

		idx := strings.IndexRune(s.Key, ':')
		req.Header.Add("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		req.Header.Add("X-Client-Ip", s.Key[:idx])

		modReqBuffer := bytes.NewBuffer(nil)
		if err := req.Write(modReqBuffer); err != nil {
			Log.Error.Print("Pcap: ", err)
			return nil, err
		}

		reqArray = append(reqArray, modReqBuffer)
	}

	return reqArray, nil
}

func seqNum(p *pcap.Packet) uint32 {
	tcpHeader := p.Headers[1].(*pcap.Tcphdr)
	return tcpHeader.Seq
}

func (s *stream) processPacket(p *Pcap, pkt *pcap.Packet) {
	// We only care about packets with content
	if len(pkt.Payload) > 0 {
		// DEBUG--->

		tcpHeader := pkt.Headers[1].(*pcap.Tcphdr)
		headerString := fmt.Sprintf("TCP: %#v", tcpHeader)
		p.EnqueueMessage(core.NewMessage(p, []byte(headerString), 0))

		// <---DEBUG

		if len(s.Packets) == 0 {
			b := bytes.NewBuffer(pkt.Payload)
			rs, err := s.reqsFromBuffer(b)
			if err == nil {
				p.handleReqs(rs)
			} else {
				s.Packets = append(s.Packets, pkt)
			}
		} else {
			b := bytes.NewBuffer(nil)
			s.Packets = append(s.Packets, pkt)
			// We ignore sequence number wrap around
			sort.Sort(s.Packets)
			expectedNext := seqNum(s.Packets[0])
			j := len(s.Packets)
			for i, pkt := range s.Packets {
				if seqNum(pkt) == expectedNext {
					b.Write(pkt.Payload)
					expectedNext += uint32(len(pkt.Payload))
				} else {
					// Hole in stream
					j = i
					break
				}
			}
			rs, err := s.reqsFromBuffer(b)
			if err != nil {
				s.Packets = s.Packets[j:]
				p.handleReqs(rs)
			}
			// When there are packets already in the stream, this means there is an incomplete request somewhere there
			// and that this packet might complete it
			// This might be a performance problem
			// Also seqNum wrap around is not handled
		}
	}
}

// Pcap consumer plugin
// Configuration example
//
//   - "consumer.Pcap":
//     Enable: true
//     Interface: eth0
//     Filter: "dst port 80 and dst host 127.0.0.1"
//
type Pcap struct {
	core.ConsumerBase
	Interface   string
	Filter      string
	Capturing   bool
	Handle      *pcap.Pcap
	Openstreams map[string]*stream
	seqNum      uint64
}

func init() {
	shared.RuntimeType.Register(Pcap{})
}

func (p *Pcap) Configure(conf core.PluginConfig) error {
	err := p.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	p.Interface = conf.GetString("Interface", "eth0")
	p.Filter = conf.GetString("Filter", "dst port 80 and dst host 127.0.0.1")
	p.Capturing = true
	p.Openstreams = make(map[string]*stream)

	return nil
}

func (p *Pcap) handleReqs(rs []*bytes.Buffer) {
	for _, r := range rs {
		p.Enqueue(r.Bytes(), p.seqNum)
		atomic.AddUint64(&p.seqNum, 1)
	}
}

func (p *Pcap) readPackets() {
	for pkt, r := p.Handle.NextEx(); p.Capturing && r >= 0; pkt, r = p.Handle.NextEx() {
		if r == 0 {
			continue // timeout, continue
		}
		pkt.Decode()
		if key, err := getKey(pkt); err == nil {
			stream, ok := p.Openstreams[key]
			if !ok {
				stream = newStream(key, p)
				p.Openstreams[key] = stream
			} else {
				// Reset timer
				stream.Timer.Reset(streamTimeOut)
			}
			stream.processPacket(p, pkt)
		} else {
			Log.Error.Print("Pcap: ", err)
		}
	}
	p.Handle.Close()
	p.WorkerDone()
}

func (p *Pcap) close() {
	p.Capturing = false
}

func (p *Pcap) Consume(workers *sync.WaitGroup) {
	// Start listening
	// device, snaplen, promisc, read timeout ms
	var err error
	p.Handle, err = pcap.OpenLive(p.Interface, int32(1<<16), true, 500)
	if err != nil {
		Log.Error.Print("Pcap: ", err)
		return
	}

	err = p.Handle.SetFilter(p.Filter)
	if err != nil {
		p.Handle.Close()
		Log.Error.Print("Pcap: ", err)
		return
	}

	p.AddMainWorker(workers)
	go p.readPackets()
	defer p.close()

	p.DefaultControlLoop(nil)
}
