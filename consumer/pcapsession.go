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
	"github.com/miekg/pcap"
	"github.com/trivago/gollum/core/log"
	"net/http"
	"strconv"
	"time"
)

type pcapSession struct {
	key     uint64
	client  string
	packets packetList
	timer   *time.Timer
	buffer  *bytes.Buffer
}

type packetList []*pcap.Packet

const seqHigh = uint32(0xFFFF0000)
const seqLow = uint32(0x0000FFFF)

func ipFromPcap(pkt *pcap.Packet) (*pcap.Iphdr, bool) {
	header, isValid := pkt.Headers[0].(*pcap.Iphdr)
	return header, isValid
}

func tcpFromPcap(pkt *pcap.Packet) (*pcap.Tcphdr, bool) {
	header, isValid := pkt.Headers[1].(*pcap.Tcphdr)
	return header, isValid
}

func (list packetList) insert(newPkt *pcap.Packet) packetList {
	if len(list) == 0 {
		return packetList{newPkt}
	}

	// 32-Bit integer overflow handling (stable for 65k blocks).
	// If the smallest (leftmost) number in the stored packages is much smaller
	// or much larger than the incoming number there is an overflow

	TCPHeader, _ := tcpFromPcap(newPkt)
	newPktSeq := TCPHeader.Seq
	TCPHeader, _ = tcpFromPcap(list[0])

	if newPktSeq < seqLow && TCPHeader.Seq > seqHigh {
		// Overflow: add low value segment packet
		for i, pkt := range list {
			TCPHeader, _ = tcpFromPcap(pkt)
			if TCPHeader.Seq > seqHigh {
				continue // skip high value segment
			}
			if newPktSeq < TCPHeader.Seq {
				return append(append(list[:i], newPkt), list[i:]...)
			}
		}
	} else if newPktSeq > seqHigh && TCPHeader.Seq < seqLow {
		// Overflow: add high value segment packet
		for i, pkt := range list {
			TCPHeader, _ = tcpFromPcap(pkt)
			if newPktSeq < TCPHeader.Seq || TCPHeader.Seq < seqLow {
				return append(append(list[:i], newPkt), list[i:]...)
			}
		}
	} else {
		// Regular insert
		for i, pkt := range list {
			TCPHeader, _ = tcpFromPcap(pkt)
			if newPktSeq < TCPHeader.Seq {
				return append(append(list[:i], newPkt), list[i:]...)
			}
		}
	}

	return append(list, newPkt)
}

func newPcapSession(key uint64, client string, timeout time.Duration, sessions pcapSessionMap) *pcapSession {
	return &pcapSession{
		key:    key,
		client: client,
		timer:  time.AfterFunc(timeout, func() { delete(sessions, key) }),
	}
}

func (session *pcapSession) isComplete() (bool, int) {
	numPackets := len(session.packets)

	// Trivial cases

	switch numPackets {
	case 0:
		return false, 0
	case 1:
		return true, len(session.packets[0].Payload)
	case 2:
		TCPHeader1, _ := tcpFromPcap(session.packets[0])
		TCPHeader2, _ := tcpFromPcap(session.packets[1])
		if TCPHeader1.Seq+1 == TCPHeader2.Seq {
			return true, len(session.packets[0].Payload) + len(session.packets[1].Payload)
		}
		return false, 0
	}

	// More than 2 packets -> loop

	TCPHeader, _ := tcpFromPcap(session.packets[0])
	payloadSize := len(session.packets[0].Payload)
	prevSeq := TCPHeader.Seq

	for i := 1; i < numPackets; i++ {
		TCPHeader, _ = tcpFromPcap(session.packets[i])
		if prevSeq+1 != TCPHeader.Seq {
			return false, 0
		}
		prevSeq = TCPHeader.Seq
		payloadSize += len(session.packets[i].Payload)
	}

	return true, payloadSize
}

func (session *pcapSession) dropPackets(size int) {
	chunkSize := 0
	for i, pkt := range session.packets {
		chunkSize += len(pkt.Payload)
		if chunkSize > size {
			session.packets = session.packets[i:]
		}
	}
	session.packets = session.packets[:0]
}

func (session *pcapSession) addPacket(cons *PcapHTTP, pkt *pcap.Packet) {
	if len(pkt.Payload) == 0 {
		return //  ### return, no payload  ###
	}

	session.packets = session.packets.insert(pkt)

	if complete, size := session.isComplete(); complete {
		payload := bytes.NewBuffer(make([]byte, 0, size))
		for _, pkt := range session.packets {
			payload.Write(pkt.Payload)
		}

		payloadReader := bufio.NewReader(payload)
		for {
			request, err := http.ReadRequest(payloadReader)
			if err != nil {
				return // ### return, invalid request: packets pending? ###
			}

			request.Header.Add("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
			request.Header.Add("X-Client-Ip", session.client)

			extPayload := bytes.NewBuffer(nil)
			if err := request.Write(extPayload); err != nil {
				// Error: ignore this request
				Log.Error.Print("PcapHTTP request writer: ", err)
			} else {
				// Enqueue this request
				cons.enqueueBuffer(extPayload.Bytes())
			}

			// Remove processed packets from list
			if payload.Len() == 0 {
				session.packets = session.packets[:0]
				return // ### return, processed everything ###
			}

			numReadBytes := size - payload.Len()
			size = payload.Len()
			session.dropPackets(numReadBytes)
		}
	}
}
