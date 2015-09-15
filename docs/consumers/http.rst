Http
====

This consumer opens a http port that accepts POST requests.
Messages will be generated from the POST body.
When attached to a fuse, this consumer will return error 503 in case that fuse is burned.
See the `API documentation <http://gollum.readthedocs.org/en/latest/consumers/http.html>`_ for additional details.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**ID**
  Allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.
**Fuse**
  Defines the name of the fuse this consumer is attached to.
  When left empty no fuse is attached. This is the default value.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**Address**
  Defines the IP or DNS of the server to listen to, followed by a port. By default this is set to ":80" which listens to localhost, port 80.
**ReadTimeoutSec**
  Defines a timeout in seconds when to stop reading from a failed connection.
**WithHeaders**
  Set to false to extract the body from the http request. When set to true the whole HTTP packet will be send. By default this is set to true.
**Certificate**
  Defines a path to a root certificate file if this consumer is to handle https connections. Left empty by default (disabled).
  If a Certificate is given, a PrivatKey must be given, too.
**PrivateKey**
  Defines a path to the private key used for https connections.
  Left empty by default (disabled).
  If a Certificate is given, a PrivatKey must be given, too.

Example
-------

.. code-block:: yaml

  - "consumer.Http":
    Enable: true
    Address: ":80"
    ReadTimeoutSec: 5
    WithHeaders: false
    Certificate: ""
    PrivateKey: ""
    Stream:
        - "stdin"
        - "console"
