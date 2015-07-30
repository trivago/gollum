Http
====

This consumer opens a http port that accepts POST requests.
Messages will be generated from the POST body.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**ID**
  Allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**Address**
  Defines the IP or DNS of the server to listen to, followed by a port. By default this is set to ":80" which listens to localhost, port 80.
**ReadTimeoutSec**
  Defines a timeout in seconds when to stop reading from a failed connection.
**WithHeaders**
  Set to false to extract the body from the http request. When set to true the whole HTTP packet will be send. By default this is set to true.

Example
-------

.. code-block:: yaml

  - "consumer.Httpd":
    Enable: true
    Address: ":80"
    ReadTimeoutSec: 5
    WithHeaders: false
    Stream:
        - "stdin"
        - "console"
