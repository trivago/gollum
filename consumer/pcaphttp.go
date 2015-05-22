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
	"fmt"
	"github.com/miekg/pcap"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// PcapHTTP consumer plugin
// Configuration example
//
//   - "consumer.PcapHTTP":
//     Enable: true
//     Interface: eth0
//     Filter: "dst port 80 and dst host 127.0.0.1"
//     Promiscuous: true
//     TimeoutMs: 3000
//     DebugTCP: false
//
type PcapHTTP struct {
	core.ConsumerBase
	netInterface   string
	filter         string
	capturing      bool
	promiscuous    bool
	debugTCP       bool
	handle         *pcap.Pcap
	sessions       pcapSessionMap
	seqNum         uint64
	sessionTimeout time.Duration
}

type pcapSessionMap map[uint32]*pcapSession

const (
	pcapNextExEOF     = -2
	pcapNextExError   = -1
	pcapNextExTimeout = 0
	pcapNextExOk      = 1
)

func init() {
	shared.RuntimeType.Register(PcapHTTP{})
}

func (cons *PcapHTTP) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.netInterface = conf.GetString("Interface", "eth0")
	cons.promiscuous = conf.GetBool("Promiscuous", true)
	cons.debugTCP = conf.GetBool("DebugTCP", false)
	cons.filter = conf.GetString("Filter", "dst port 80 and dst host 127.0.0.1")
	cons.capturing = true
	cons.sessions = make(pcapSessionMap)
	cons.sessionTimeout = time.Duration(conf.GetInt("TimeoutSec", 3000)) * time.Millisecond

	return nil
}

func (cons *PcapHTTP) enqueueBuffer(data []byte) {
	cons.Enqueue(data, cons.seqNum)
	atomic.AddUint64(&cons.seqNum, 1)
}

func (cons *PcapHTTP) getStreamKey(pkt *pcap.Packet) (uint32, string, bool) {
	if len(pkt.Headers) != 2 {
		Log.Debug.Printf("Invalid number of headers: %d", len(pkt.Headers))
		Log.Debug.Printf("Not a TCP/IP packet: %#v", pkt)
		return 0, "", false
	}

	ipHeader, isIPHeader := ipFromPcap(pkt)
	tcpHeader, isTCPHeader := tcpFromPcap(pkt)
	if !isIPHeader || !isTCPHeader {
		Log.Debug.Printf("Not a TCP/IP packet: %#v", pkt)
		return 0, "", false
	}

	clientID := fmt.Sprintf("%s:%d", ipHeader.SrcAddr(), tcpHeader.SrcPort)
	key := fmt.Sprintf("%s-%s:%d", clientID, ipHeader.DestAddr(), tcpHeader.DestPort)
	keyHash := fnv.New32a()
	keyHash.Sum([]byte(key))

	return keyHash.Sum32(), clientID, true
}

func (cons *PcapHTTP) readPackets() {
	for cons.capturing {
		pkt, resultCode := cons.handle.NextEx()

		switch resultCode {
		case pcapNextExEOF:
			cons.capturing = false
			Log.Note.Print("PcapHTTP: End of file, stopping.")
			continue

		case pcapNextExError:
			Log.Error.Print("PcapHTTP: ", cons.handle.Geterror())
			continue

		case pcapNextExTimeout:
			continue
		}

		pkt.Decode()
		key, client, validPacket := cons.getStreamKey(pkt)

		if cons.debugTCP {
			TCPHeader, _ := tcpFromPcap(pkt)
			headerString := fmt.Sprintf("TCP: [%t] %#v", validPacket && len(pkt.Payload) > 0, TCPHeader)
			cons.enqueueBuffer([]byte(headerString))
		}

		if validPacket {
			session, sessionExists := cons.sessions[key]
			if sessionExists {
				session.timer.Reset(cons.sessionTimeout)
			} else {
				session = newPcapSession(client)
				cons.sessions[key] = session

				session.timer = time.AfterFunc(cons.sessionTimeout, func() {
					if len(session.packets) > 0 {
						Log.Debug.Print("PcapHTTP: Incomplete session timed out")
					}
					delete(cons.sessions, key)
				})
			}
			session.addPacket(cons, pkt)
		}
	}
	cons.handle.Close()
	cons.WorkerDone()
}

func (cons *PcapHTTP) close() {
	cons.capturing = false
}

func (cons *PcapHTTP) Consume(workers *sync.WaitGroup) {
	// Start listening
	// device, snaplen, promisc, read timeout ms
	var err error
	cons.handle, err = pcap.OpenLive(cons.netInterface, int32(1<<16), cons.promiscuous, 500)
	if err != nil {
		Log.Error.Print("PcapHTTP: ", err)
		return
	}

	err = cons.handle.SetFilter(cons.filter)
	if err != nil {
		cons.handle.Close()
		Log.Error.Print("PcapHTTP: ", err)
		return
	}

	cons.AddMainWorker(workers)
	go cons.readPackets()
	defer cons.close()

	cons.DefaultControlLoop(nil)
}
