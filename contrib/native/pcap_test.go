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

package native

import (
	"github.com/miekg/pcap"
	"github.com/trivago/gollum/shared"
	"testing"
)

func newTcpPktMock(seq uint32) *pcap.Packet {
	return &pcap.Packet{
		Headers: []interface{}{
			&pcap.Iphdr{},
			&pcap.Tcphdr{Seq: seq},
		},
		Payload: make([]byte, 1),
	}
}

func TestFindPacketSlot(t *testing.T) {
	expect := shared.NewExpect(t)

	listMock := packetList{
		newTcpPktMock(1),
		newTcpPktMock(2),
		newTcpPktMock(3),
		newTcpPktMock(4),
		newTcpPktMock(6),
		newTcpPktMock(7),
		newTcpPktMock(8),
		newTcpPktMock(9),
		newTcpPktMock(11),
		newTcpPktMock(12),
		newTcpPktMock(15),
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
	expect := shared.NewExpect(t)

	listMock := packetList{
		newTcpPktMock(1),
		newTcpPktMock(2),
		newTcpPktMock(3),
		newTcpPktMock(4),
		newTcpPktMock(6),
		newTcpPktMock(7),
		newTcpPktMock(8),
		newTcpPktMock(9),
		newTcpPktMock(11),
		newTcpPktMock(12),
		newTcpPktMock(14),
	}

	listMock = listMock.insert(newTcpPktMock(0))
	listMock = listMock.insert(newTcpPktMock(13))
	listMock = listMock.insert(newTcpPktMock(15))
	listMock = listMock.insert(newTcpPktMock(5))
	listMock = listMock.insert(newTcpPktMock(10))

	complete, _ := listMock.isComplete()
	expect.True(complete)

	listMock = listMock.insert(newTcpPktMock(0xFFFFFFFF))
	listMock = listMock.insert(newTcpPktMock(0xFFFFFFFE))
	listMock = listMock.insert(newTcpPktMock(16))

	tcpHeader, _ := tcpFromPcap(listMock[0])
	expect.Equal(uint32(0xFFFFFFFE), tcpHeader.Seq)

	tcpHeader, _ = tcpFromPcap(listMock[1])
	expect.Equal(uint32(0xFFFFFFFF), tcpHeader.Seq)

	tcpHeader, _ = tcpFromPcap(listMock[len(listMock)-1])
	expect.Equal(uint32(16), tcpHeader.Seq)

	complete, _ = listMock.isComplete()
	expect.True(complete)
}
