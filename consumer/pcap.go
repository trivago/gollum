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
	"errors"
	"fmt"
	"github.com/miekg/pcap"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Pcap consumer plugin
// Configuration example
//
//   - "consumer.Pcap":
//     Enable: true
//     Intf: eth0
//     Port: 80
//
// This consumer does not define any options beside the standard ones.
type Stream struct {
	Key     string
	Timer   *time.Timer
	Packets StreamPackets
	Buf     bytes.Buffer
}

const streamTimeOut = 3 * time.Second

func getKey(p *pcap.Packet) (string, error) {
	if len(p.Headers) == 2 {
		h0 := p.Headers[0].(*pcap.Iphdr)
		h1 := p.Headers[1].(*pcap.Tcphdr)
		s := fmt.Sprintf("%s:%d-%s:%d", h0.SrcAddr(), h1.SrcPort, h0.DestAddr(), h1.DestPort)
		return s, nil
	}
	return "", errors.New("Invalid Packet")
}

func (s *Stream) reqsFromBuffer(b *bytes.Buffer) (rs []*bytes.Buffer, err error) {
	r := bufio.NewReader(b)
	for !readerDone(r) {
		req, err := http.ReadRequest(r)
		if err == nil {
			req.Header.Add("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
			idx := strings.IndexRune(s.Key, ':')
			req.Header.Add("X-Client-Ip", s.Key[:idx])
			resBuf := bytes.NewBuffer(nil)
			err = req.Write(resBuf)
			if err != nil {
				goto Error
			}
			rs = append(rs, resBuf)
		} else {
			goto Error
		}
	}
	return
Error:
	rs = nil
	fmt.Println(err.Error())
	return
}

func (s *Stream) processPacket(p *Pcap, pkt *pcap.Packet) {
	// We only care about packets with content
	if len(pkt.Payload) > 0 {
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

func seqNum(p *pcap.Packet) uint32 {
	h := p.Headers[1].(*pcap.Tcphdr)
	return h.Seq
}

type StreamPackets []*pcap.Packet

func (a StreamPackets) Len() int           { return len(a) }
func (a StreamPackets) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StreamPackets) Less(i, j int) bool { return seqNum(a[i]) < seqNum(a[j]) }

type Pcap struct {
	core.ConsumerBase
	Intf        string
	Port        int
	Capturing   bool
	Handle      *pcap.Pcap
	OpenStreams map[string]*Stream
	GSeq        uint64
}

func init() {
	shared.RuntimeType.Register(Pcap{})
}

func (p *Pcap) Close() {
	p.Capturing = false
	p.Handle.Close()
	p.WorkerDone()
}

func (p *Pcap) Configure(conf core.PluginConfig) error {
	err := p.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	p.Intf = conf.GetString("Intf", "eth0")
	p.Port = conf.GetInt("Port", 80)
	p.Capturing = true
	p.OpenStreams = make(map[string]*Stream)

	// device, snaplen, promisc, read timeout ms
	p.Handle, err = pcap.OpenLive(p.Intf, int32(1<<16), true, 500)
	if p.Handle == nil {
		return err
	}
	expr := fmt.Sprintf("dst port %d", p.Port)

	err = p.Handle.SetFilter(expr)
	if err != nil {
		return err
	}
	return nil
}

// A small hack to check if there is more to be read
func readerDone(r *bufio.Reader) bool {
	_, e := r.Peek(1)
	return e != nil
}

func (p *Pcap) handleReqs(rs []*bytes.Buffer) {
	for _, r := range rs {
		p.Enqueue(r.Bytes(), p.GSeq)
		p.GSeq++
	}
}

func (p *Pcap) closeStream(s *Stream) {
	// Stream closing...
	delete(p.OpenStreams, s.Key)
}

func (p *Pcap) newStream(k string, pkt *pcap.Packet) *Stream {
	s := &Stream{}
	s.Key = k
	s.Timer = time.AfterFunc(streamTimeOut, func() { p.closeStream(s) })
	s.Packets = make([]*pcap.Packet, 0)
	return s
}

func (p *Pcap) readPackets() {
	for pkt, r := p.Handle.NextEx(); p.Capturing && r >= 0; pkt, r = p.Handle.NextEx() {
		if r == 0 {
			// timeout, continue
			continue
		}
		pkt.Decode()
		if k, err := getKey(pkt); err == nil {
			stream, ok := p.OpenStreams[k]
			if !ok {
				stream = p.newStream(k, pkt)
				p.OpenStreams[k] = stream
			} else {
				// Reset timer
				stream.Timer.Reset(streamTimeOut)
			}
			stream.processPacket(p, pkt)
		} else {
			fmt.Println(err.Error())
		}
	}
}

func (p *Pcap) Consume(workers *sync.WaitGroup) {
	p.AddMainWorker(workers)
	go p.readPackets()
	defer p.Close()
	p.DefaultControlLoop(nil)
}
