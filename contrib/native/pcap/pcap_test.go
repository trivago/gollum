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
	"testing"

	"github.com/miekg/pcap"
	"github.com/trivago/tgo/ttesting"
)

func newTCPPktMock(seq uint32) *pcap.Packet {
	return &pcap.Packet{
		Headers: []interface{}{
			&pcap.Iphdr{},
			&pcap.Tcphdr{Seq: seq},
		},
		Payload: make([]byte, 1),
	}
}

func TestFindPacketSlot(t *testing.T) {
	expect := ttesting.NewExpect(t)

	listMock := packetList{
		newTCPPktMock(1),
		newTCPPktMock(2),
		newTCPPktMock(3),
		newTCPPktMock(4),
		newTCPPktMock(6),
		newTCPPktMock(7),
		newTCPPktMock(8),
		newTCPPktMock(9),
		newTCPPktMock(11),
		newTCPPktMock(12),
		newTCPPktMock(15),
	}

	// Duplicate
	idx, duplicate := findPacketSlot(1, listMock)
	expect.True(duplicate)
	expect.Equal(0, idx)

	// Front
	idx, duplicate = findPacketSlot(0, listMock)
	expect.False(duplicate)
	expect.Equal(0, idx)

	// Back
	idx, duplicate = findPacketSlot(17, listMock)
	expect.False(duplicate)
	expect.Equal(11, idx)

	// middle-left
	idx, duplicate = findPacketSlot(5, listMock)
	expect.False(duplicate)
	expect.Equal(4, idx)

	// middle-right
	idx, duplicate = findPacketSlot(10, listMock)
	expect.False(duplicate)
	expect.Equal(8, idx)
}

func TestInsert(t *testing.T) {
	expect := ttesting.NewExpect(t)

	listMock := packetList{
		newTCPPktMock(1),
		newTCPPktMock(2),
		newTCPPktMock(3),
		newTCPPktMock(4),
		newTCPPktMock(6),
		newTCPPktMock(7),
		newTCPPktMock(8),
		newTCPPktMock(9),
		newTCPPktMock(11),
		newTCPPktMock(12),
		newTCPPktMock(14),
	}

	listMock = listMock.insert(newTCPPktMock(0))
	listMock = listMock.insert(newTCPPktMock(13))
	listMock = listMock.insert(newTCPPktMock(15))
	listMock = listMock.insert(newTCPPktMock(5))
	listMock = listMock.insert(newTCPPktMock(10))

	complete, _ := listMock.isComplete()
	expect.True(complete)

	listMock = listMock.insert(newTCPPktMock(0xFFFFFFFF))
	listMock = listMock.insert(newTCPPktMock(0xFFFFFFFE))
	listMock = listMock.insert(newTCPPktMock(16))

	tcpHeader, _ := tcpFromPcap(listMock[0])
	expect.Equal(uint32(0xFFFFFFFE), tcpHeader.Seq)

	tcpHeader, _ = tcpFromPcap(listMock[1])
	expect.Equal(uint32(0xFFFFFFFF), tcpHeader.Seq)

	tcpHeader, _ = tcpFromPcap(listMock[len(listMock)-1])
	expect.Equal(uint32(16), tcpHeader.Seq)

	complete, _ = listMock.isComplete()
	expect.True(complete)
}
