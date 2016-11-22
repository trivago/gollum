Kinesis
=======

This consumer reads message from an AWS Kinesis stream.
When attached to a fuse, this consumer will stop processing messages in case that fuse is burned.


Parameters
----------

**Enable**
  Enable switches the consumer on or off.
  By default this value is set to true.

**ID**
  ID allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.

**Stream**
  Stream contains either a single string or a list of strings defining the message channels this consumer will produce.
  By default this is set to "*" which means only producers set to consume "all streams" will get these messages.

**Fuse**
  Fuse defines the name of a fuse to observe for this consumer.
  Producer may "burn" the fuse when they encounter errors.
  Consumers may react on this by e.g. closing connections to notify any writing services of the problem.
  Set to "" by default which disables the fuse feature for this consumer.
  It is up to the consumer implementation to react on a broken fuse in an appropriate manner.

**KinesisStream**
  KinesisStream defines the stream to read from.
  By default this is set to "default".

**Region**
  Region defines the amazon region of your kinesis stream.
  By default this is set to "eu-west-1".

**Endpoint**
  Endpoint defines the amazon endpoint for your kinesis stream.
  By default this is et to "kinesis.eu-west-1.amazonaws.com".

**CredentialType**
  CredentialType defines the credentials that are to be used when connectiong to kensis.
  This can be one of the following: environment, static, shared, none.
  Static enables the parameters CredentialId, CredentialToken and CredentialSecretm shared enables the parameters CredentialFile and CredentialProfile.
  None will not use any credentials and environment will pull the credentials from environmental settings.
  By default this is set to none.

**DefaultOffset**
  DefaultOffset defines the message index to start reading from.
  Valid values are either "Newset", "Oldest", or a number.
  The default value is "Newest".

**OffsetFile**
  OffsetFile defines a file to store the current offset per shard.
  By default this is set to "", i.e. it is disabled.
  If a file is set and found consuming will start after the stored offset.

**RecordsPerQuery**
  RecordsPerQuery defines the number of records to pull per query.
  By default this is set to 100.

**RecordMessageDelimiter**
  RecordMessageDelimiter defines the string to delimit messages within a record.
  By default this is set to "", i.e. it is disabled.

**QuerySleepTimeMs**
  QuerySleepTimeMs defines the number of milliseconds to sleep before trying to pull new records from a shard that did not return any records.
  By default this is set to 1000.

**RetrySleepTimeSec**
  RetrySleepTimeSec defines the number of seconds to wait after trying to reconnect to a shard.
  By default this is set to 4.

Example
-------

.. code-block:: yaml

	- "consumer.Kinesis":
	    Enable: true
	    ID: ""
	    Fuse: ""
	    Stream:
	        - "foo"
	        - "bar"
	    KinesisStream: "default"
	    Region: "eu-west-1"
	    Endpoint: "kinesis.eu-west-1.amazonaws.com"
	    DefaultOffset: "Newest"
	    OffsetFile: ""
	    RecordsPerQuery: 100
	    RecordMessageDelimiter: ""
	    QuerySleepTimeMs: 1000
	    RetrySleepTimeSec: 4
	    CredentialType: "none"
	    CredentialId: ""
	    CredentialToken: ""
	    CredentialSecret: ""
	    CredentialFile: ""
	    CredentialProfile: ""
