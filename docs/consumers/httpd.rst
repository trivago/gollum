Httpd
#############

This consumer opens a http port that accepts POST requests.
Messages will be generated from the POST body.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**Address**
  Defines the IP or DNS of the server to listen to, followed by a port. By default this is set to ":80" which listens to localhost, port 80.
**ReadTimeoutSec**
  Defines a timeout in seconds when to stop reading from a failed connection.

Example
-------

::

  - "consumer.Httpd":
    Enable: true
    Address: ":80"
    ReadTimeoutSec: 5
    Stream:
        - "stdin"
        - "console"
