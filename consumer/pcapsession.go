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
	// "github.com/miekg/pcap"
	"encoding/binary"
	"github.com/fmardini/pcap"
	"github.com/trivago/gollum/core/log"
	"io"
	"net/http"
	"strconv"
	"time"
)

type pcapSession struct {
	client    string
	timer     *time.Timer
	packets   packetList
	lastError error
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

// create a new TCP session buffer
func newPcapSession(client string) *pcapSession {
	return &pcapSession{
		client: client,
	}
}

func checksum(pkt *pcap.Packet) bool {
	ipH, _ := ipFromPcap(pkt)
	tcpH, _ := tcpFromPcap(pkt)
	var res uint32
	res += uint32(binary.BigEndian.Uint16(ipH.SrcIp[:2]))
	res += uint32(binary.BigEndian.Uint16(ipH.SrcIp[2:]))
	res += uint32(binary.BigEndian.Uint16(ipH.DestIp[:2]))
	res += uint32(binary.BigEndian.Uint16(ipH.DestIp[2:]))
	res += 6 // Protocol
	var l uint32 = uint32(tcpH.DataOffset*4) + uint32(len(pkt.Payload))
	res += l

	for i := uint32(0); i < l-1; i += 2 {
		if i == 16 {
			continue
		}
		res += uint32(binary.BigEndian.Uint16(tcpH.Data[i : i+2]))
	}
	if l&0x01 != 0 {
		tb := make([]byte, 2)
		tb[0] = tcpH.Data[l-1]
		res += uint32(binary.BigEndian.Uint16(tb))
	}
	for (res >> 16) > 0 {
		res = (res & 0xFFFF) + (res >> 16)
	}
	cksum := uint16(^res & 0xFFFF)

	return cksum == tcpH.Checksum
}

// binary search for an insertion point in the packet list
func findPacketSlot(seq uint32, list packetList) (int, bool) {
	offset := 0
	partLen := len(list)

	for partLen > 1 {
		median := partLen / 2
		TCPHeader, _ := tcpFromPcap(list[offset+median])
		switch {
		case seq < TCPHeader.Seq:
			partLen = median
		case seq > TCPHeader.Seq:
			offset += median + 1
			partLen -= median + 1
		default:
			return offset, true // ### return, duplicate ###
		}
	}

	if partLen == 0 {
		return offset, false // ### return, out of range ###
	}

	TCPHeader, _ := tcpFromPcap(list[offset])
	if seq > TCPHeader.Seq {
		return offset + 1, false
	}

	return offset, seq == TCPHeader.Seq // ### return, duplicate ###
}

// insertion sort for new packages by sequence number
func (list packetList) insert(newPkt *pcap.Packet) packetList {
	if len(list) == 0 {
		return packetList{newPkt}
	}

	// 32-Bit integer overflow handling (stable for 65k blocks).
	// If the smallest (leftmost) number in the stored packages is much smaller
	// or much larger than the incoming number there is an overflow

	TCPHeader, _ := tcpFromPcap(newPkt)
	newPktSeq := TCPHeader.Seq
	frontTCPHeader, _ := tcpFromPcap(list[0])
	backTCPHeader, _ := tcpFromPcap(list[len(list)-1])

	if newPktSeq < seqLow && frontTCPHeader.Seq > seqHigh {
		// Overflow: add low value segment packet
		for i, pkt := range list {
			TCPHeader, _ = tcpFromPcap(pkt)
			if TCPHeader.Seq > seqHigh {
				continue // skip high value segment
			}
			if newPktSeq < TCPHeader.Seq {
				return append(list[:i], append(packetList{newPkt}, list[i:]...)...) // ### return insert ###
			}
			if newPktSeq == TCPHeader.Seq {
				list[i] = newPkt
				return list // ### return, replaced ###
			}
		}
	} else if newPktSeq > seqHigh && backTCPHeader.Seq < seqLow {
		// Overflow: add high value segment packet
		for i, pkt := range list {
			TCPHeader, _ = tcpFromPcap(pkt)
			if newPktSeq < TCPHeader.Seq || TCPHeader.Seq < seqLow {
				return append(list[:i], append(packetList{newPkt}, list[i:]...)...) // ### return insert ###
			}
			if newPktSeq == TCPHeader.Seq {
				list[i] = newPkt
				return list // ### return, replaced ###
			}
		}
	} else {
		// Large package insert (binary search)
		if len(list) > 10 {
			i, duplicate := findPacketSlot(newPktSeq, list)
			switch {
			case duplicate:
				list[i] = newPkt
				return list // ### return, replaced ###
			case i < 0:
				return append(packetList{newPkt}, list...) // ### return, prepend ###
			case i < len(list):
				return append(list[:i], append(packetList{newPkt}, list[i:]...)...) // ### return insert ###
			}
		} else {
			// Small package insert (linear Search)
			for i, pkt := range list {
				TCPHeader, _ = tcpFromPcap(pkt)
				if newPktSeq < TCPHeader.Seq {
					return append(list[:i], append(packetList{newPkt}, list[i:]...)...) // ### return insert ###
				}
				if newPktSeq == TCPHeader.Seq {
					list[i] = newPkt
					return list // ### return, replaced ###
				}
			}
		}
	}

	return append(list, newPkt)
}

// check if all packets have consecutive sequence numbers and calculate the
// buffer size required for processing
func (list packetList) isComplete() (bool, int) {
	numPackets := len(list)

	// Trivial cases

	switch numPackets {
	case 0:
		return false, 0
	case 1:
		return true, len(list[0].Payload)
	case 2:
		TCPHeader0, _ := tcpFromPcap(list[0])
		TCPHeader1, _ := tcpFromPcap(list[1])
		size0 := len(list[0].Payload)

		if TCPHeader0.Seq+uint32(size0) == TCPHeader1.Seq {
			return true, size0 + len(list[1].Payload)
		}

		return false, 0
	}

	// More than 2 packets -> loop

	TCPHeader, _ := tcpFromPcap(list[0])
	payloadSize := len(list[0].Payload)
	prevSeq := TCPHeader.Seq
	prevSize := len(list[0].Payload)

	for i := 1; i < numPackets; i++ {
		TCPHeader, _ = tcpFromPcap(list[i])

		if prevSeq+uint32(prevSize) != TCPHeader.Seq {
			return false, 0
		}

		prevSeq = TCPHeader.Seq
		prevSize = len(list[i].Payload)
		payloadSize += prevSize
	}

	return true, payloadSize
}

func (session *pcapSession) String() string {
	info := fmt.Sprintf("%d:{", len(session.packets))
	for _, pkt := range session.packets {
		tcpHeader, _ := tcpFromPcap(pkt)
		info += fmt.Sprintf("[%d]", tcpHeader.Seq)
	}
	return info + "}"
}

// remove processed packets from the list.
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

// add a TCP packet to the session and try to generate a HTTP packet
func (session *pcapSession) addPacket(cons *PcapHTTP, pkt *pcap.Packet) {
	if len(pkt.Payload) == 0 {
		return //  ### return, no payload  ###
	}

	session.packets = session.packets.insert(pkt)

	if complete, size := session.packets.isComplete(); complete {
		payload := bytes.NewBuffer(make([]byte, 0, size))
		for _, pkt := range session.packets {
			payload.Write(pkt.Payload)
		}

		payloadReader := bufio.NewReader(payload)
		for {
			request, err := http.ReadRequest(payloadReader)
			if err != nil {
				session.lastError = err
				return // ### return, invalid request: packets pending? ###
			}

			request.Header.Add("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
			request.Header.Add("X-Client-Ip", session.client)

			extPayload := bytes.NewBuffer(nil)
			if err := request.Write(extPayload); err != nil {
				if err == io.ErrUnexpectedEOF {
					return // ### return, invalid request: packets pending? ###
				}
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
