// Copyright 2015-2017 trivago GmbH
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

package native

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/miekg/pcap"
	"github.com/trivago/gollum/core"
)

// PcapHTTPConsumer consumer plugin
// NOTICE: This producer is not included in standard builds. To enable it
// you need to trigger a custom build with native plugins enabled.
// This plugin utilizes libpcap to listen for network traffic and reassamble
// http requests from it. As it uses a CGO based library it will break cross
// platform builds (i.e. you will have to compile it on the correct platform).
// Configuration example
//
//  - "native.PcapHTTPConsumer":
//    Enable: true
//    Interface: eth0
//    Filter: "dst port 80 and dst host 127.0.0.1"
//    Promiscuous: true
//    TimeoutMs: 3000
//
// Interface defines the network interface to listen on. By default this is set
// to eth0, get your specific value from ifconfig.
//
// Filter defines a libpcap filter for the incoming packages. You can filter for
// specific ports, portocols, ips, etc.. The documentation can be found here:
// http://www.tcpdump.org/manpages/pcap-filter.7.txt (manpage).
// By default this is set to listen on port 80 for localhost packages.
//
// Promiscuous switches the network interface defined by Interface into
// promiscuous mode. This is required if you want to listen for all packages
// coming from the network, even those that were not meant for the ip bound
// to the interface you listen on. Enabling this can increase your CPU load.
// This setting is enabled by default.
//
// TimeoutMs defines a timeout after which a tcp session is considered to have
// sent to the fallback, i.e. the (remaining) packages will be discarded. Every incoming
// packet will restart the timer for the specific client session.
// By default this is set to 3000, i.e. 3 seconds.
type PcapHTTPConsumer struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	netInterface        string
	filter              string
	promiscuous         bool
	handle              *pcap.Pcap
	sessions            pcapSessionMap
	sessionTimeout      time.Duration
	sessionGuard        *sync.Mutex
}

type pcapSessionMap map[uint32]*pcapSession

const (
	pcapNextExEOF     = -2
	pcapNextExError   = -1
	pcapNextExTimeout = 0
	pcapNextExOk      = 1
	pcapFin           = 0x1
)

func init() {
	core.TypeRegistry.Register(PcapHTTPConsumer{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *PcapHTTPConsumer) Configure(conf core.PluginConfigReader) error {
	cons.netInterface = conf.GetString("Interface", "eth0")
	cons.promiscuous = conf.GetBool("Promiscuous", true)
	cons.filter = conf.GetString("Filter", "dst port 80 and dst host 127.0.0.1")
	cons.sessions = make(pcapSessionMap)
	cons.sessionTimeout = time.Duration(conf.GetInt("TimeoutMs", 3000)) * time.Millisecond
	cons.sessionGuard = new(sync.Mutex)

	return conf.Errors.OrNil()
}

func (cons *PcapHTTPConsumer) enqueueBuffer(data []byte) {
	cons.Enqueue(data)
}

func (cons *PcapHTTPConsumer) getStreamKey(pkt *pcap.Packet) (uint32, string, bool) {
	if len(pkt.Headers) != 2 {
		cons.Logger.Debugf("Invalid number of headers: %d", len(pkt.Headers))
		cons.Logger.Debug("Not a TCP/IP packet: %#v", pkt)
		return 0, "", false
	}

	ipHeader, isIPHeader := ipFromPcap(pkt)
	tcpHeader, isTCPHeader := tcpFromPcap(pkt)
	if !isIPHeader || !isTCPHeader {
		cons.Logger.Debugf("Not a TCP/IP packet: %#v", pkt)
		return 0, "", false
	}

	if len(pkt.Payload) == 0 {
		return 0, "", false
	}

	clientID := fmt.Sprintf("%s:%d", ipHeader.SrcAddr(), tcpHeader.SrcPort)
	key := fmt.Sprintf("%s-%s:%d", clientID, ipHeader.DestAddr(), tcpHeader.DestPort)
	keyHash := fnv.New32a()
	keyHash.Write([]byte(key))

	return keyHash.Sum32(), clientID, true
}

func (cons *PcapHTTPConsumer) readPackets() {
	defer func() {
		cons.handle.Close()
		if panicMessage := recover(); panicMessage != nil {
			// try again
			cons.Logger.Error("[PANIC] PcapHTTPConsumer: ", panicMessage)
			cons.initPcap()
			go cons.readPackets()
		} else {
			// done
			cons.WorkerDone()
		}
	}()

	for cons.IsActive() {
		pkt, resultCode := cons.handle.NextEx()

		switch resultCode {
		case pcapNextExEOF:
			cons.Control() <- core.PluginControlStopConsumer
			cons.Logger.Info("PcapHTTPConsumer: End of file, stopping.")
			continue

		case pcapNextExError:
			cons.Logger.Error("PcapHTTPConsumer: ", cons.handle.Geterror())
			continue

		case pcapNextExTimeout:
			continue
		}

		pkt.Decode()
		TCPHeader, _ := tcpFromPcap(pkt)

		key, client, validPacket := cons.getStreamKey(pkt)
		session, sessionExists := cons.tryGetSession(key)

		if validPacket {
			if sessionExists {
				session.timer.Reset(cons.sessionTimeout)
			} else {
				session = newPcapSession(client, cons.Logger)
				cons.setSession(key, session)

				session.timer = time.AfterFunc(cons.sessionTimeout, func() {
					if len(session.packets) > 0 {
						if session.lastError == nil {
							session.lastError = fmt.Errorf("-")
						}
						// TODO: Try to recover io.ErrUnexpectedEOF by adding \r\n
						//       Generate the missing package (seq + payload)
						cons.Logger.Debugf("PcapHTTPConsumer: Incomplete session 0x%X timed out: \"%s\" %s", key, session.lastError, session)
					}
					cons.clearSession(key)
				})
			}
			session.addPacket(cons, pkt)
		}

		if sessionExists && validPacket && (TCPHeader.Flags&pcapFin != 0) {
			closeString := fmt.Sprintf("TCP: [closed] [%s]", client)
			cons.enqueueBuffer([]byte(closeString))

			session.timer.Stop()
			cons.clearSession(key)
		}
	}
}

func (cons *PcapHTTPConsumer) tryGetSession(key uint32) (*pcapSession, bool) {
	cons.sessionGuard.Lock()
	defer cons.sessionGuard.Unlock()
	session, exists := cons.sessions[key]
	return session, exists
}

func (cons *PcapHTTPConsumer) setSession(key uint32, session *pcapSession) {
	cons.sessionGuard.Lock()
	defer cons.sessionGuard.Unlock()
	cons.sessions[key] = session
}

func (cons *PcapHTTPConsumer) clearSession(key uint32) {
	cons.sessionGuard.Lock()
	defer cons.sessionGuard.Unlock()
	delete(cons.sessions, key)
}

func (cons *PcapHTTPConsumer) initPcap() {
	var err error

	// Start listening
	// device, snaplen, promisc, read timeout ms
	cons.handle, err = pcap.OpenLive(cons.netInterface, int32(1<<16), cons.promiscuous, 500)
	if err != nil {
		cons.Logger.Error("PcapHTTPConsumer: ", err)
		return
	}

	err = cons.handle.SetFilter(cons.filter)
	if err != nil {
		cons.handle.Close()
		cons.Logger.Error("PcapHTTPConsumer: ", err)
		return
	}
}

// Consume enables libpcap monitoring as configured.
func (cons *PcapHTTPConsumer) Consume(workers *sync.WaitGroup) {
	cons.initPcap()
	cons.AddMainWorker(workers)

	go cons.readPackets()
	cons.ControlLoop()
}
