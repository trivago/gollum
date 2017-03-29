# pcap

## pcaphttp.go

This plugin utilizes libpcap to listen for network traffic and reassamble
http requests from it. As it uses a CGO based library it will break cross
platform builds (i.e. you will have to compile it on the correct platform).

Interface defines the network interface to listen on. By default this is set
to eth0, get your specific value from ifconfig.

### Requirements

## pcapsession.go

### Requirements

