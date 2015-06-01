HttpReq
=======

This producer sends messages that already are valid http request to a given webserver.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**Stream**
  Defines either one or an aray of stream names this producer recieves messages from.
**Address**
  Defines the server address to connect to.
  This can be any ip address and port like "localhost:5880". By default this is set to ":80".

Example
-------

.. code-block:: yaml

  - "producer.HttpReq":
    Enable: true
    Address: "testing:80"
    Stream: "http"
