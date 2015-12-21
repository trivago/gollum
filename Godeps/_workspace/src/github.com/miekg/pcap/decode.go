package pcap

import (
	"fmt"
	"net"
	"strings"
)

const (
	TYPE_IP  = 0x0800
	TYPE_ARP = 0x0806
	TYPE_IP6 = 0x86DD

	IP_ICMP = 1
	IP_INIP = 4
	IP_TCP  = 6
	IP_UDP  = 17
)

// Port from sf-pcap.c file.
const (
	TCPDUMP_MAGIC           = 0xa1b2c3d4
	KUZNETZOV_TCPDUMP_MAGIC = 0xa1b2cd34
	FMESQUITA_TCPDUMP_MAGIC = 0xa1b234cd
	NAVTEL_TCPDUMP_MAGIC    = 0xa12b3c4d
	NSEC_TCPDUMP_MAGIC      = 0xa1b23c4d
)

// DLT,
// these are the types that are the same on all platforms, and that
// have been defined by <net/bpf.h> for ages.
const (
	DLT_NULL    = 0  // BSD loopback encapsulation
	DLT_EN10MB  = 1  // Ethernet (10Mb)
	DLT_EN3MB   = 2  // Experimental Ethernet (3Mb)
	DLT_AX25    = 3  // Amateur Radio AX.25
	DLT_PRONET  = 4  // Proteon ProNET Token Ring
	DLT_CHAOS   = 5  // Chaos
	DLT_IEEE802 = 6  // 802.5 Token Ring
	DLT_ARCNET  = 7  // ARCNET, with BSD-style header
	DLT_SLIP    = 8  // Serial Line IP
	DLT_PPP     = 9  // Point-to-point Protocol
	DLT_FDDI    = 10 // FDDI
)

const (
	ERRBUF_SIZE = 256

	// According to pcap-linktype(7).
	LINKTYPE_NULL       = DLT_NULL
	LINKTYPE_ETHERNET   = DLT_EN10MB
	LINKTYPE_TOKEN_RING = DLT_IEEE802

	LINKTYPE_EXP_ETHERNET = DLT_EN3MB /* 3Mb experimental Ethernet */
	LINKTYPE_AX25         = DLT_AX25
	LINKTYPE_PRONET       = DLT_PRONET
	LINKTYPE_CHAOS        = DLT_CHAOS
	LINKTYPE_ARCNET_BSD   = DLT_ARCNET /* BSD-style headers */
	LINKTYPE_SLIP         = DLT_SLIP
	LINKTYPE_PPP          = DLT_PPP
	LINKTYPE_FDDI         = DLT_FDDI

	LINKTYPE_ARCNET           = 7
	LINKTYPE_ATM_RFC1483      = 100
	LINKTYPE_RAW              = 101
	LINKTYPE_PPP_HDLC         = 50
	LINKTYPE_PPP_ETHER        = 51
	LINKTYPE_C_HDLC           = 104
	LINKTYPE_IEEE802_11       = 105
	LINKTYPE_FRELAY           = 107
	LINKTYPE_LOOP             = 108
	LINKTYPE_LINUX_SLL        = 113
	LINKTYPE_LTALK            = 104
	LINKTYPE_PFLOG            = 117
	LINKTYPE_PRISM_HEADER     = 119
	LINKTYPE_IP_OVER_FC       = 122
	LINKTYPE_SUNATM           = 123
	LINKTYPE_IEEE802_11_RADIO = 127
	LINKTYPE_ARCNET_LINUX     = 129
	LINKTYPE_LINUX_IRDA       = 144
	LINKTYPE_LINUX_LAPD       = 177
)

type addrHdr interface {
	SrcAddr() string
	DestAddr() string
	Len() int
}

type addrStringer interface {
	String(addr addrHdr) string
}

func decodemac(pkt []byte) uint64 {
	mac := uint64(0)
	for i := uint(0); i < 6; i++ {
		mac = (mac << 8) + uint64(pkt[i])
	}
	return mac
}

// Arphdr is a ARP packet header.
type Arphdr struct {
	Addrtype          uint16
	Protocol          uint16
	HwAddressSize     uint8
	ProtAddressSize   uint8
	Operation         uint16
	SourceHwAddress   []byte
	SourceProtAddress []byte
	DestHwAddress     []byte
	DestProtAddress   []byte
}

func (arp *Arphdr) String() (s string) {
	switch arp.Operation {
	case 1:
		s = "ARP request"
	case 2:
		s = "ARP Reply"
	}
	if arp.Addrtype == LINKTYPE_ETHERNET && arp.Protocol == TYPE_IP {
		s = fmt.Sprintf("%012x (%s) > %012x (%s)",
			decodemac(arp.SourceHwAddress), arp.SourceProtAddress,
			decodemac(arp.DestHwAddress), arp.DestProtAddress)
	} else {
		s = fmt.Sprintf("addrtype = %d protocol = %d", arp.Addrtype, arp.Protocol)
	}
	return
}

// IPhdr is the header of an IP packet.
type Iphdr struct {
	Version    uint8
	Ihl        uint8
	Tos        uint8
	Length     uint16
	Id         uint16
	Flags      uint8
	FragOffset uint16
	Ttl        uint8
	Protocol   uint8
	Checksum   uint16
	SrcIp      []byte
	DestIp     []byte
}

func (ip *Iphdr) SrcAddr() string  { return net.IP(ip.SrcIp).String() }
func (ip *Iphdr) DestAddr() string { return net.IP(ip.DestIp).String() }
func (ip *Iphdr) Len() int         { return int(ip.Length) }

type Tcphdr struct {
	SrcPort    uint16
	DestPort   uint16
	Seq        uint32
	Ack        uint32
	DataOffset uint8
	Flags      uint16
	Window     uint16
	Checksum   uint16
	Urgent     uint16
	Data       []byte
}

const (
	TCP_FIN = 1 << iota
	TCP_SYN
	TCP_RST
	TCP_PSH
	TCP_ACK
	TCP_URG
	TCP_ECE
	TCP_CWR
	TCP_NS
)

func (tcp *Tcphdr) String(hdr addrHdr) string {
	return fmt.Sprintf("TCP %s:%d > %s:%d %s SEQ=%d ACK=%d LEN=%d",
		hdr.SrcAddr(), int(tcp.SrcPort), hdr.DestAddr(), int(tcp.DestPort),
		tcp.FlagsString(), int64(tcp.Seq), int64(tcp.Ack), hdr.Len())
}

func (tcp *Tcphdr) FlagsString() string {
	var sflags []string
	if 0 != (tcp.Flags & TCP_SYN) {
		sflags = append(sflags, "syn")
	}
	if 0 != (tcp.Flags & TCP_FIN) {
		sflags = append(sflags, "fin")
	}
	if 0 != (tcp.Flags & TCP_ACK) {
		sflags = append(sflags, "ack")
	}
	if 0 != (tcp.Flags & TCP_PSH) {
		sflags = append(sflags, "psh")
	}
	if 0 != (tcp.Flags & TCP_RST) {
		sflags = append(sflags, "rst")
	}
	if 0 != (tcp.Flags & TCP_URG) {
		sflags = append(sflags, "urg")
	}
	if 0 != (tcp.Flags & TCP_NS) {
		sflags = append(sflags, "ns")
	}
	if 0 != (tcp.Flags & TCP_CWR) {
		sflags = append(sflags, "cwr")
	}
	if 0 != (tcp.Flags & TCP_ECE) {
		sflags = append(sflags, "ece")
	}
	return fmt.Sprintf("[%s]", strings.Join(sflags, " "))
}

type Udphdr struct {
	SrcPort  uint16
	DestPort uint16
	Length   uint16
	Checksum uint16
}

func (udp *Udphdr) String(hdr addrHdr) string {
	return fmt.Sprintf("UDP %s:%d > %s:%d LEN=%d CHKSUM=%d",
		hdr.SrcAddr(), int(udp.SrcPort), hdr.DestAddr(), int(udp.DestPort),
		int(udp.Length), int(udp.Checksum))
}

type Icmphdr struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	Id       uint16
	Seq      uint16
	Data     []byte
}

func (icmp *Icmphdr) String(hdr addrHdr) string {
	return fmt.Sprintf("ICMP %s > %s Type = %d Code = %d ",
		hdr.SrcAddr(), hdr.DestAddr(), icmp.Type, icmp.Code)
}

func (icmp *Icmphdr) TypeString() (result string) {
	switch icmp.Type {
	case 0:
		result = fmt.Sprintf("Echo reply seq=%d", icmp.Seq)
	case 3:
		switch icmp.Code {
		case 0:
			result = "Network unreachable"
		case 1:
			result = "Host unreachable"
		case 2:
			result = "Protocol unreachable"
		case 3:
			result = "Port unreachable"
		default:
			result = "Destination unreachable"
		}
	case 8:
		result = fmt.Sprintf("Echo request seq=%d", icmp.Seq)
	case 30:
		result = "Traceroute"
	}
	return
}

type Ip6hdr struct {
	// http://www.networksorcery.com/enp/protocol/ipv6.htm
	Version      uint8  // 4 bits
	TrafficClass uint8  // 8 bits
	FlowLabel    uint32 // 20 bits
	Length       uint16 // 16 bits
	NextHeader   uint8  // 8 bits, same as Protocol in Iphdr
	HopLimit     uint8  // 8 bits
	SrcIp        []byte // 16 bytes
	DestIp       []byte // 16 bytes
}

func (ip6 *Ip6hdr) SrcAddr() string  { return net.IP(ip6.SrcIp).String() }
func (ip6 *Ip6hdr) DestAddr() string { return net.IP(ip6.DestIp).String() }
func (ip6 *Ip6hdr) Len() int         { return int(ip6.Length) }
