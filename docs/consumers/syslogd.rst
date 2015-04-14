Syslogd
#############

This consumer opens up a syslogd compatible socket.

Parameters
----------

**Enable**
    Can either be true or false to enable or disable this consumer.
**Stream**
    Defines either one or an aray of stream names this consumer sends messages to.
**Address**
  Defines the protocol, address/DNS and port to listen to.
  The protocol can either be "socket://" for unix domain, "tcp://" for TCP or "udp://" for UDP sockets.
  Please note that the format used may overwrite this setting if a certain protocol is enforced by it.
  Set to "udp://0.0.0.0:514" by default.
**Format**
  Supports one of three formats ("RFC6587" by default):

  - `RFC3164 <https://tools.ietf.org/html/rfc3164>`_ udp only
  - `RFC5424 <https://tools.ietf.org/html/rfc5424>`_ udp only
  - `RFC6587 <https://tools.ietf.org/html/rfc6587>`_ tcp or udp

Example
-------

::

  - "consumer.Syslogd":
    Enable: true
    Address: "socket://var/run/gollum_syslogd.socket"
    Format: "RFC6587"
    Stream: "log"
