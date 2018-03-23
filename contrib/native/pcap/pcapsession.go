// Copyright 2015-2018 trivago N.V.
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

// +build cgo,!unit

package native

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/miekg/pcap"
	"github.com/sirupsen/logrus"
)

type pcapSession struct {
	client    string
	timer     *time.Timer
	packets   packetList
	lastError error
	logger    logrus.FieldLogger
}

type packetList []*pcap.Packet

const seqHigh = uint32(0xFFFFFF00)
const seqLow = uint32(0x00FFFFFF)

func ipFromPcap(pkt *pcap.Packet) (*pcap.Iphdr, bool) {
	header, isValid := pkt.Headers[0].(*pcap.Iphdr)
	return header, isValid
}

func tcpFromPcap(pkt *pcap.Packet) (*pcap.Tcphdr, bool) {
	header, isValid := pkt.Headers[1].(*pcap.Tcphdr)
	return header, isValid
}

// create a new TCP session buffer
func newPcapSession(client string, logger logrus.FieldLogger) *pcapSession {
	return &pcapSession{
		client: client,
		logger: logger,
	}
}

func validateChecksum(pkt *pcap.Packet) bool {
	ipHeader, _ := ipFromPcap(pkt)
	tcpHeader, _ := tcpFromPcap(pkt)
	dataLength := uint32(tcpHeader.DataOffset*4) + uint32(len(pkt.Payload))

	checksum := uint32(0)
	checksum += uint32(binary.BigEndian.Uint16(ipHeader.SrcIp[:2]))
	checksum += uint32(binary.BigEndian.Uint16(ipHeader.SrcIp[2:]))
	checksum += uint32(binary.BigEndian.Uint16(ipHeader.DestIp[:2]))
	checksum += uint32(binary.BigEndian.Uint16(ipHeader.DestIp[2:]))
	checksum += 6 // Protocol is always TCP
	checksum += dataLength

	for i := uint32(0); i < dataLength-1; i += 2 {
		if i == 16 {
			continue
		}
		checksum += uint32(binary.BigEndian.Uint16(tcpHeader.Data[i : i+2]))
	}
	if dataLength&0x01 != 0 {
		paddedBuffer := make([]byte, 2)
		paddedBuffer[0] = tcpHeader.Data[dataLength-1]
		checksum += uint32(binary.BigEndian.Uint16(paddedBuffer))
	}
	for (checksum >> 16) > 0 {
		checksum = (checksum & 0xFFFF) + (checksum >> 16)
	}

	crc16 := uint16(^checksum & 0xFFFF)
	return crc16 == tcpHeader.Checksum
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
	info := fmt.Sprintf("{")
	for _, pkt := range session.packets {
		tcpHeader, _ := tcpFromPcap(pkt)
		info += fmt.Sprintf("0x%X:0x%X, ", tcpHeader.Seq, tcpHeader.Seq+uint32(len(pkt.Payload)))
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
func (session *pcapSession) addPacket(cons *PcapHTTPConsumer, pkt *pcap.Packet) {
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
					session.lastError = err
					return // ### return, invalid request: packets pending? ###
				}
				// Error: ignore this request
				session.logger.Error("PcapHTTPConsumer request writer: ", err)
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
