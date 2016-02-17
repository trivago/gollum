// Package pcap is a wrapper around the pcap library.
package pcap

/*
#cgo LDFLAGS: -lpcap
#include <stdlib.h>
#include <pcap.h>

// Workaround for not knowing how to cast to const u_char**
int hack_pcap_next_ex(pcap_t * p, struct pcap_pkthdr **pkt_header,
		      u_char ** pkt_data)
{
	return pcap_next_ex(p, pkt_header, (const u_char **)pkt_data);
}

void hack_pcap_dump(pcap_dumper_t * dumper, struct pcap_pkthdr *pkt_header,
		      u_char * pkt_data)
{
    pcap_dump((u_char *)dumper, pkt_header, pkt_data);
}
*/
import "C"

import (
	"errors"
	"net"
	"syscall"
	"time"
	"unsafe"
)

type Pcap struct {
	cptr *C.pcap_t
}

type PcapDumper struct {
	cptr *C.pcap_dumper_t
}

type pcapError struct{ string }

type Stat struct {
	PacketsReceived  uint32
	PacketsDropped   uint32
	PacketsIfDropped uint32
}

type Interface struct {
	Name        string
	Description string
	Addresses   []IFAddress
	// TODO: add more elements
}

type IFAddress struct {
	IP      net.IP
	Netmask net.IPMask
	// TODO: add broadcast + PtP dst ?
}

func Version() string               { return C.GoString(C.pcap_lib_version()) }
func (p *Pcap) Datalink() int       { return int(C.pcap_datalink(p.cptr)) }
func (e *pcapError) Error() string  { return e.string }
func (p *Pcap) Geterror() error     { return &pcapError{C.GoString(C.pcap_geterr(p.cptr))} }
func (p *Pcap) Next() (pkt *Packet) { rv, _ := p.NextEx(); return rv }

func Create(device string) (handle *Pcap, err error) {
	var buf *C.char
	buf = (*C.char)(C.calloc(ERRBUF_SIZE, 1))
	h := new(Pcap)

	dev := C.CString(device)
	defer C.free(unsafe.Pointer(dev))

	h.cptr = C.pcap_create(dev, buf)
	if nil == h.cptr {
		handle = nil
		err = &pcapError{C.GoString(buf)}
	} else {
		handle = h
	}

	C.free(unsafe.Pointer(buf))
	return
}

// Set buffer size (units in bytes) on activated handle.
func (p *Pcap) SetBufferSize(sz int32) error {
	if C.pcap_set_buffer_size(p.cptr, C.int(sz)) != 0 {
		return p.Geterror()
	}
	return nil
}

//  If arg p is non-zero promiscuous mode will be set on capture handle when it is activated.
func (p *Pcap) SetPromisc(promisc bool) error {
	var pro int32
	if promisc {
		pro = 1
	}

	if C.pcap_set_promisc(p.cptr, C.int(pro)) != 0 {
		return p.Geterror()
	}
	return nil
}

func (p *Pcap) SetSnapLen(s int32) error {
	if C.pcap_set_snaplen(p.cptr, C.int(s)) != 0 {
		return p.Geterror()
	}
	return nil
}

// Set read timeout (milliseconds) that will be used on a capture handle when it is activated.
func (p *Pcap) SetReadTimeout(toMs int32) error {
	if C.pcap_set_timeout(p.cptr, C.int(toMs)) != 0 {
		return p.Geterror()
	}
	return nil
}

// Activate a packet capture handle to look at packets on the network, with the options that
// were set on the handle being in effect.
func (p *Pcap) Activate() error {
	if C.pcap_activate(p.cptr) != 0 {
		return p.Geterror()
	}
	return nil
}

// OpenLive opens a device and returns a handler.
func OpenLive(device string, snaplen int32, promisc bool, timeout_ms int32) (handle *Pcap, err error) {
	var buf *C.char
	buf = (*C.char)(C.calloc(ERRBUF_SIZE, 1))
	h := new(Pcap)
	var pro int32
	if promisc {
		pro = 1
	}

	dev := C.CString(device)
	defer C.free(unsafe.Pointer(dev))

	h.cptr = C.pcap_open_live(dev, C.int(snaplen), C.int(pro), C.int(timeout_ms), buf)
	if nil == h.cptr {
		handle = nil
		err = &pcapError{C.GoString(buf)}
	} else {
		handle = h
	}
	C.free(unsafe.Pointer(buf))
	return
}

// Openoffline
func OpenOffline(file string) (handle *Pcap, err error) {
	var buf *C.char
	buf = (*C.char)(C.calloc(ERRBUF_SIZE, 1))
	h := new(Pcap)

	cf := C.CString(file)
	defer C.free(unsafe.Pointer(cf))

	h.cptr = C.pcap_open_offline(cf, buf)
	if nil == h.cptr {
		handle = nil
		err = &pcapError{C.GoString(buf)}
	} else {
		handle = h
	}
	C.free(unsafe.Pointer(buf))
	return
}

// Pcap closes a handler.
func (p *Pcap) Close() {
	C.pcap_close(p.cptr)
}

func (p *Pcap) NextEx() (pkt *Packet, result int32) {
	var pkthdr_ptr *C.struct_pcap_pkthdr
	var pkthdr C.struct_pcap_pkthdr

	var buf_ptr *C.u_char
	var buf unsafe.Pointer
	result = int32(C.hack_pcap_next_ex(p.cptr, &pkthdr_ptr, &buf_ptr))

	buf = unsafe.Pointer(buf_ptr)
	pkthdr = *pkthdr_ptr

	if nil == buf {
		return
	}
	pkt = new(Packet)
	pkt.Time = time.Unix(int64(pkthdr.ts.tv_sec), int64(pkthdr.ts.tv_usec)*1000) // pcap provides usec but time.Unix requires nsec
	pkt.Caplen = uint32(pkthdr.caplen)
	pkt.Len = uint32(pkthdr.len)
	pkt.Data = make([]byte, pkthdr.caplen)

	for i := uint32(0); i < pkt.Caplen; i++ {
		pkt.Data[i] = *(*byte)(unsafe.Pointer(uintptr(buf) + uintptr(i)))
	}
	return
}

func (p *Pcap) Getstats() (stat *Stat, err error) {
	var cstats _Ctype_struct_pcap_stat
	if -1 == C.pcap_stats(p.cptr, &cstats) {
		return nil, p.Geterror()
	}
	stats := new(Stat)
	stats.PacketsReceived = uint32(cstats.ps_recv)
	stats.PacketsDropped = uint32(cstats.ps_drop)
	stats.PacketsIfDropped = uint32(cstats.ps_ifdrop)

	return stats, nil
}

func (p *Pcap) SetFilter(expr string) (err error) {
	var bpf _Ctype_struct_bpf_program
	cexpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cexpr))

	if -1 == C.pcap_compile(p.cptr, &bpf, cexpr, 1, 0) {
		return p.Geterror()
	}

	if -1 == C.pcap_setfilter(p.cptr, &bpf) {
		C.pcap_freecode(&bpf)
		return p.Geterror()
	}
	C.pcap_freecode(&bpf)
	return nil
}

func (p *Pcap) SetDirection(direction string) (err error) {
	var pcap_direction C.pcap_direction_t
	if direction == "in" {
		pcap_direction = C.PCAP_D_IN
	} else if direction == "out" {
		pcap_direction = C.PCAP_D_OUT
	} else {
		pcap_direction = C.PCAP_D_INOUT
	}
	if -1 == C.pcap_setdirection(p.cptr, pcap_direction) {
		return p.Geterror()
	}
	return nil
}

func (p *Pcap) SetDataLink(dlt int) error {
	if -1 == C.pcap_set_datalink(p.cptr, C.int(dlt)) {
		return p.Geterror()
	}
	return nil
}

func DatalinkValueToName(dlt int) string {
	if name := C.pcap_datalink_val_to_name(C.int(dlt)); name != nil {
		return C.GoString(name)
	}
	return ""
}

func DatalinkValueToDescription(dlt int) string {
	if desc := C.pcap_datalink_val_to_description(C.int(dlt)); desc != nil {
		return C.GoString(desc)
	}
	return ""
}

func FindAllDevs() (ifs []Interface, err error) {
	var buf *C.char
	buf = (*C.char)(C.calloc(ERRBUF_SIZE, 1))
	defer C.free(unsafe.Pointer(buf))
	var alldevsp *C.pcap_if_t

	if -1 == C.pcap_findalldevs((**C.pcap_if_t)(&alldevsp), buf) {
		return nil, errors.New(C.GoString(buf))
	}
	defer C.pcap_freealldevs((*C.pcap_if_t)(alldevsp))
	dev := alldevsp
	var i uint32
	for i = 0; dev != nil; dev = (*C.pcap_if_t)(dev.next) {
		i++
	}
	ifs = make([]Interface, i)
	dev = alldevsp
	for j := uint32(0); dev != nil; dev = (*C.pcap_if_t)(dev.next) {
		var iface Interface
		iface.Name = C.GoString(dev.name)
		iface.Description = C.GoString(dev.description)
		iface.Addresses = findAllAddresses(dev.addresses)
		// TODO: add more elements
		ifs[j] = iface
		j++
	}
	return
}

func findAllAddresses(addresses *_Ctype_struct_pcap_addr) (retval []IFAddress) {
	// TODO - make it support more than IPv4 and IPv6?
	retval = make([]IFAddress, 0, 1)
	for curaddr := addresses; curaddr != nil; curaddr = (*_Ctype_struct_pcap_addr)(curaddr.next) {
		if curaddr.addr == nil {
			continue
		}
		var a IFAddress
		var err error
		if a.IP, err = sockaddrToIP((*syscall.RawSockaddr)(unsafe.Pointer(curaddr.addr))); err != nil {
			continue
		}
		if a.Netmask, err = sockaddrToIP((*syscall.RawSockaddr)(unsafe.Pointer(curaddr.netmask))); err != nil {
			continue
		}
		retval = append(retval, a)
	}
	return
}

func sockaddrToIP(rsa *syscall.RawSockaddr) (IP []byte, err error) {
	switch rsa.Family {
	case syscall.AF_INET:
		pp := (*syscall.RawSockaddrInet4)(unsafe.Pointer(rsa))
		IP = make([]byte, 4)
		for i := 0; i < len(IP); i++ {
			IP[i] = pp.Addr[i]
		}
		return
	case syscall.AF_INET6:
		pp := (*syscall.RawSockaddrInet6)(unsafe.Pointer(rsa))
		IP = make([]byte, 16)
		for i := 0; i < len(IP); i++ {
			IP[i] = pp.Addr[i]
		}
		return
	}
	err = errors.New("Unsupported address type")
	return
}

// Inject ...
func (p *Pcap) Inject(data []byte) (err error) {
	buf := (*C.char)(C.malloc((C.size_t)(len(data))))

	for i := 0; i < len(data); i++ {
		*(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(buf)) + uintptr(i))) = data[i]
	}

	if -1 == C.pcap_inject(p.cptr, unsafe.Pointer(buf), (C.size_t)(len(data))) {
		err = p.Geterror()
	}
	C.free(unsafe.Pointer(buf))
	return
}

func (p *Pcap) DumpOpen(ofile *string) (dumper *PcapDumper, err error) {
	d := new(PcapDumper)
	d.cptr = C.pcap_dump_open(p.cptr, C.CString(*ofile))
	if nil == d.cptr {
		return d, errors.New("Cannot open dumpfile")
	}
	dumper = d
	return
}

func (p *Pcap) PcapLoop(pktnum int, dumper *PcapDumper) (result int32, err error) {
	var pkthdr_ptr *C.struct_pcap_pkthdr
	var buf_ptr *C.u_char

	for i := 0; true; {
		result = int32(C.hack_pcap_next_ex(p.cptr, &pkthdr_ptr, &buf_ptr))
		switch result {
		case 0:
			continue // timeout
		case 1:
			// success : capturing packet
		case -1:
			return result, errors.New("Error in pcap next ex")
		case -2:
			return // reach EOF in offline mode
		}
		if nil == buf_ptr {
			continue
		}
		if nil != dumper {
			p.PcapDump(dumper, pkthdr_ptr, buf_ptr)
			p.PcapDumpFlush(dumper)
		}
		if pktnum > 0 {
			i++
			if i >= pktnum {
				break
			}
		}
	}
	return
}

func (p *Pcap) PcapDump(dumper *PcapDumper, pkthdr_ptr *C.struct_pcap_pkthdr, buf_ptr *C.u_char) {
	C.hack_pcap_dump(dumper.cptr, pkthdr_ptr, buf_ptr)
}

func (p *Pcap) PcapDumpFlush(dumper *PcapDumper) error {
	if -1 == C.pcap_dump_flush(dumper.cptr) {
		return p.Geterror()
	}
	return nil
}

func (p *Pcap) PcapDumpClose(dumper *PcapDumper) {
	C.pcap_dump_close(dumper.cptr)
}
