# Gollum changelog

## 0.6.0

Gollum 0.6.0 contains breaking changes over version 0.5.x.
Please read the release notes carefully

Gollum 0.6.0 dependency management switches from go-dep to go-modules.
As of this is recommended to use go 1.11 or later for development.
Go 1.10 and 1.9 are still supported. Support for Go 1.8 or older has been dropped.

### New with 0.6.0

* Added a new flag "-mt" to choose the metrics provider (currently only prometheus).
* Consumer.File setting "Files" now supports glob patterns.
* Consumer.Syslog now allows non-standard protocol types (see issue #234)
* Message metadata can now store arbitrary data
* When not setting the numbers of CPU, gollum will try to use cgroup limits
* format.Cast for changing metadata field types
* format.Override to set static field values

### Breaking changes with 0.6.0

* Format.SplitPick default delimiter is now ","
* Multiple formatters have been renamed to support the new metadata model
* Metrics are now collected using go-metrics. This allows e.g. prometheus output (default).
  Old-style metrics have been removed and many metrics names have changed.
* Consumer.File setting "File" has been renamed to "Files"
* Consumer.File setting "OffsetFile" changed to "OffsetPath" to support multiple offset files per consumer.
* Consumer.File setting "PollingDelay" has been renamed to "PollingDelayMs".
* Metadata type has changed from `map[string][]byte` to `tgo.MarshalMap`.
* Deserializing messages written by v0.5.x will lead to metadata of those message to be discarded.
* Removed support for go 1.8 in order to allow sync.Map
* The functions Message.ResizePayload and .ExtendPayload have been removed in favor if go's slice internal functions.

## 0.5.4

This is a patch / minor features release.

### Fixed with 0.5.4

* producer.spooling is now functional again as messages were not written correctly since 0.5.0 (#248).
* producer.spooling now does not block upon shutdown (#248).
* metadata is now handled correctly when messages are sent to fallback (#247).
* producer.socket now sends messages directly to fallback if connect fails.
* 
## 0.5.3

This is a patch / minor features release.

### Fixed with 0.5.3

* Fixed a GC panic/crash caused by tgo.ByteBuffer.

## 0.5.2

This is a patch / minor features release.

### New with 0.5.2

* The version number is now generated via make and git. This will properly identify versions between releases.
* New producer.AwsCloudwatchLogs. Thanks to @luqasz
* The makefile has been cleaned up and go meta-linter support has been added

### Fixed with 0.5.2

* consumer.Kafka now properly commits the consumer offsets to kafka. Thanks to @crewton
* producer.awsKinesis failed to produce records under certain conditions
* The consumer.Kafka folderPermissions property is now correctly applied
* formt.ExtractJSON trimValues property is now correctly applied
* The gollum binary inside the Dockerfile is built on the same baseimage as deployed
* Filter will now always filter out the MODIFIED message, not the original. This behavior is more "expected".

## 0.5.1

This is a patch / minor features release.

### New with 0.5.1

* format.MetadataCopy has been updated to support free copying between metadata and payload
* producer.ElasticSearch alles setting the format of timeBasedIndex
* format.GrokToJSON has new options: RemoveEmptyValues, NamedCapturesOnly and SkipDefaultPatterns
* Using dep for dependencies instead of glide

### Fixed with 0.5.1

* Fixed inversion of -lc always
* Fixed a nil pointer panic with producer.elasticsearch when receiving messages with unassigned streams
* producer.ElasticSearch settings are now named according to config
* producer.ElasticSearch dayBasedIndex renamed to timeBasedIndex and it's now working as expected
* Updated dependencies to latest version (brings support for kafka 1.0, fixes user agent parsing for format.processTSV)

## 0.5.0

Gollum 0.5.0 contains major breaking changes in all areas.
Configuration files working with Gollum 0.4.x will not work with this verison unless changed.
Please have a look at the [transition guide](http://gollum.readthedocs.io/en/latest/src/releaseNotes/v0.5.0.html#breaking-changes-0-4-x-to-0-5-0) for details.

Important note:
When switching a pipline from 0.4.x to 0.5.0, make sure all spooling data has been read.
Messages serialized to disk with 0.4.x are not compatible with 0.5.0.

### New with 0.5.0

* Filters and Formatters have been merged into one list
* You can now use a filter or formatter more than once in the same plugin
* Consumers can now do filtering and formatting, too
* Messages can now store metadata. Formatters can affect the payload or a metadata field
* All plugins now have an automatic log scope
* Message payloads are now backed by a memory pool
* Messages now store the original message, i.e. a backup of the payload state after consumer processing
* Gollum now provides per-stream metrics
* Plugins are now able to implement health checks that can be queried via http
* New base types for producers: Direct, Buffered, Batched
* Plugin configurations now support nested structures
* The configuration process has been simplified a lot by adding automatic error handling and struct tags
* All plugin configuration keys are now case insensitive
* Added a new formatter format.GrokToJSON
* Added a new formatter format.JSONToInflux10
* Added a new formatter format.Double
* Added a new formatter format.MetadataCopy
* Added a new formatter format.Trim
* Consumer.File now supports filesystem events
* Consumers can now define the number of go routines used for formatting/filtering
* All AWS plugins now support role switching
* All AWS plugins are now based on the same credentials code

### Fixed with 0.5.0

* The plugin lifecycle has been reimplemented to avoid gollum being stuck waiting for plugins to change state
* Integration test suite added
* Producer.HTTPRequest port handling fixed
* The test-config command will now produce more meaningful results
* Duplicating messages now properly duplicates the whole message and not just the struct
* Several race conditions have been fixed
* Producer.ElasticSearch is now based on a more up-to-date library
* Producer.AwsS3 is now behaving more like producer.File
* Gollum metrics can now bind to a specific address instead of just a port

### Breaking changes with 0.5.0

* The config format has changed to improve automatic processing
* A lot of plugins have been renamed to avoid confusion and to better reflect their behavior
* A lot of plugins parameters have been renamed
* The instances plugin parameter has been removed
* Most of gollum's metrics have been renamed
* Plugin base types have been renamed
* All message handling function signatures have changed to use pointers
* All formatters don't daisy chain anymore as they can now be listed in proper order
* Stream plugins have been renamed to Router plugins
* Routers are not allowed to modify message content anymore
* filter.All and format.Forward have been removed as they are not required anymore
* Producer formatter listss dedicated to format a key or similar constructs have been removed
* Logging framework switched to logrus
* The package gollum.shared has been removed in favor of trivago.tgo
* Fuses have been removed from all plugins
* The general message sequence number has been removed
* The term "drop" has been replaced by the term "fallback" to emphasise it's use
* The \_DROPPED\_ stream has been removed. Messages are discarded if no fallback is set
* Formatters can still the stream of a message but cannot trigger routing by themselves
* Compiling contrib plugins now requires a specific loader.go to be added
* The docker file on docker hub is now a lot smaller and only contains the gollum binary
* The message serialization format has been changed

## 0.4.5

This is a patch / minor features release.
All vendor dependencies have been updated to the latest version and binaries have been compiled with go 1.8.

### Fixed with 0.4.5

* producer.Kafka will discard messages returned as "too large" to avoid spooling
* consumer.HTTP does not truncate messages with WithHeaders:false anymore (thanks @mhils)
* producer.Websocket now uses gorilla websockets (thanks @glaslos)
* Dockerfile is now working again
* It is now possible to (optionally) send nil messages with producer.kafka again
* Consumer.Kinesis will renew the iterator object when hitting a timeout
* Consumer.Kinesis now runs with an offset file set that does not exist
* Consumer.Kinesis offset file is now written less often (after each batch)
* Consumer.Kafka does now retry with an "oldest" offset after encountering an OutOfRange exception.
* Fixed a crash when using producer.ElasticSearch with date based indexes (thanks @relud)
* format.Base64Decode now uses data from previous formatters as intended
* format.JSON arr and obj will now auto create a key if necessary
* format.JSON now checks for valid state references upon startup
* format.JSON now properly encodes strings when using "enc"
* format.SplitToJSON may now keep JSON payload and is better at escaping string
* "gollum -tc" will exit with error code 1 upon error
* "gollum -tc" will now properly display errors during config checking

### New with 0.4.5

* Added producer for writing data to Amazon S3 (thanks @relud)
* Added authentication support to consumer.HTTP (thanks @glaslos)
* Added authentication support to native.KafkaProducer (thanks @relud)
* Added authentication support to producer.Kafka (thanks @relud)
* Added authentication support to consumer.Kafka (thanks @relud)
* Added consumer group support to consumer.Kafka (thanks @relud)
* Added a native SystemD consumer (thanks @relud)
* Added a Statsd producer for counting messages (thanks @relud)
* Added an option to flatten JSON arrays into single values with format.ProcessJSON (thanks @relud)
* Added filter.Any to allow "or" style combinations of filters (thanks @relud)
* Added support for unix timestamp parsing to format.ProcessJSON (thanks @relud)
* Added filter.Sample to allow processing of every n'th message only (thanks @relud)
* Added format.TemplateJSON to apply golang templates to JSON payloads (thanks @relud)
* Added named pipe support to consumer.Console
* Added "pick" option to format.ProcessJSON to get a single value from an arrays
* Extended "remove" option pf format.ProcessJSON to remove values from arrays
* Added "geoip" option to format.ProcessJSON to get GeoIP data from an IP
* Added index configuration options to producer.ElasticSearch

## 0.4.4

This is a patch / minor features release.
All vendor dependencies have been updated to the latest version and binaries have been compiled with go 1.7.

### Fixed with 0.4.4

* Fixed file offset handling in consumer.Kinesis (thanks @relud)
* Fixed producer.File RotatePruneAfterHours setting
* Producer.File symlink switch is now atomic
* Fixed panic in producer.Redis when Formatter was not set
* Fixed producer.Spooling being stuck for a long time during shutdown
* Fixed native.KafkaProducer to map all topics to "default" if no topic mapping was set
* Fixed a concurrent map write during initialization in native.KafkaProducer
* Fixed consumer.Kafka OffsetFile setting stopping gollum when the offset file was not present
* consumer.Kafka will retry to connect to a not (yet) existing topic every PersistTimeoutMs
* Consumer.Kafka now tries to connect every ServerTimeoutSec if initial connect fails
* Consumer.Kafka MessageBufferCount default value increased to 8192
* Producer.Kafka and native.KafkaProducer now discard messages with 0-Byte content
* Producer.Kafka SendRetries set to 1 by default to circumvent a reconnect issue within sarama
* Fixed panic in producer.Kafka when shutting down
* Added manual heartbeat to check a valid broker connection with producer.Kafka
* Format.Base64Encode now returns the original message if decoding failed
* socket.producer TCP can be used without ACK
* Consumer.Syslogd message handling differences between RFC3164 and RFC5424 / RFC6587 fixed

### New with 0.4.4

* New AWS Firehose producer (thanks @relud)
* New format.ProcessTSV for modifying TSV encoded messages (thanks @relud)
* Added user agent parsing to format.ProcessJSON (thanks @relud)
* Added support for unix timestamp parsing to format.ProcessJSON (thanks @relud)
* Added support for new shard detection to consumer.Kinesis (thanks @relud)
* Added support for mulitple messages per record to producer.Kinesis and consumer.Kinesis (thanks @relud)
* Added "remove" directive for format.ProcessJSON
* Added key Formatter support for producer.Redis
* Added RateLimited- metrics for filter.Rate
* Added format.Clear to remove message content (e.g. useful for key formatters)
* Added "KeyFormatterFirst" for producer.Kafka and native.KafkaProducer
* Added Version support for producer.Kafka and consumer.Kafka
* Added ClientID support for consumer.Kafka
* Added folder creation capatibilites to consumer.File when creating offset files
* Added gollum log messages metrics
* Added wildcard topic mapping to producer.Kafka and native.KafkaProducer
* Added FilterAfterFormat to producer.Kafka and native.KafkaProducer
* Producer.Spooling now continuously looks for new streams to read
* Producer.Spooling now reacts on SIGHUP to trigger a respooling
* Seperated version information to -r (version, go runtime, modules) and -v (just version) command line flag
* Added trace commandline flag

## 0.4.3

This is a patch / minor features release.
It includes several configuration changes for producer.Kafka that might change the runtime behavior.
Please check your configuration files.

### Fixed with 0.4.3

* Fixed several race conditions reported by Go 1.6 and go build -race
* Fixed the scribe producer to drop unformatted message in case of error
* Fixed file.consumer rotation to work on regular files, too
* Fixed file.consumer rotation to reset the offset file after a SIGHUP
* Dockerfiles updated
* Producer.Kafka now sends messages directly to avoid sarama performance bottlenecks
* consumer.Kafka offset file is properly read on startup if DefaultOffset "oldest" or "newest" is
* Exisiting unix domain socket detection changed to use create instead of stat (better error handling)
* Kafka and Scribe specific metrics are now updated if there are no messages, too
* Scribe producer is now reacting better to server connection errors
* Filters and Formatters are now covered with unittests

### New with 0.4.3

* Support for Go1.5 vendor experiment
* New producer for librdkafka (not included in standard builds)
* Metrics added to show memory consumption
* New kafka metrics added to show "roundtrip" times for messages
* producer.Benchmark added to get more meaningful core system profiling results
* New filter filter.Rate added to allow limiting streams to a certain number of messages per second
* Added key support to consumer.Kafka and producer.Kafka
* Added an "ordered read" config option to consumer.Kafka (round robin reading)
* Added a new formater format.ExtractJSON to extract a single value from a JSON object
* Go version is now printed with gollum -v
* Scribe producer now queries scribe server status in regular intervals
* format.Sequence separator character can now be configured
* format.Runlength separator character can now be configured

### Other changes with 0.4.3

* Renamed producer.Kafka BatchTimeoutSec to BatchTimeoutMs to allow millisecond based values
* producer.Kafka retry count set to 0
* producer.Kafka default producer set to RoundRobin
* producer.Kafka GracePeriodMs default set to 100
* producer.Kafka MetadataRefreshMs default set to 600000 (10 minutes)
* producer.Kafka TimeoutMs default set to 10000 (10 seconds)
* filter.RegExp FilterExpressionNot is evaluated before FilterExpression
* filter.RegExp FilterExpression is evaluated if FilterExpressionNot passed

## 0.4.2

This is a patch / minor features release.

### Fixed with 0.4.2

* consumer.SysLogD now has more meaningful errormessages
* consumer.File now properly supports file rotation if the file to read is a symlink
* Scribe and Kafka metrics are now only updated upon successful send
* Fixed an out of bounds panic when producer.File was rotating logfiles without an extension
* Compression of files after rotation by produer.File now works (again)
* producer.Kafka now only reconnects if all topics report an error
* producer.Spool now properly respools long messages
* producer.Spool will not delete a file if a message in it could not be processed
* producer.Spool will try to automatically respool files after a restart
* producer.Spool will rotate non-empty files even if no new messages come in
* producer.Spool will recreate folders when removed during runtime
* producer.Spool will drop messages if rotation failes (not reroute)
* Messages that are spooled twice now retain their original stream
* Better handling of situations where Sarama (Kafka) writes become blocking
* Plugins now start up as "initializing" not as "dead" preventing dropped messages during startup

### New with 0.4.2

* New formatter format.SplitToJSON to convert CSV data to JSON
* New formatter format.ProcessJSON to modify JSON data
* producer.File can now set permissions for any folders created
* RPM spec file added
* producer.File can now add zero padding to rotated file numbering
* producer.File can now prune logfiles by file age
* producer.Spool can now be rate limited
* Dev version (major.minor.patch.dev) is now part of the metrics
* New AWS Kinesis producer and consumer (thanks @relud)

## 0.4.1

This is a patch / minor features release

### Fixed with 0.4.1

* InfluxDB JSON and line protocol fixed
* shared.WaitGroup.WaitFor with duration 0 falls back to shared.WaitGroup.Wait
* proper io.EOF handling for shared.BufferedReader and shared.ByteStream
* HTTP consumer now responds with 200 instead of 203
* HTTP consumer properly handles EOF
* Increased test coverage

### New with 0.4.1

* Support for InfluxDB line protocol
* New setting to enable/disable InfluxDB time based database names
* Introduction of "fuses" (circuit breaker pattern)
* Added HTTPs support for HTTP consumer
* Added POST data support to HTTPRequest producer

## 0.4.0

This release includes several reliability fixes that prevent messages from being lost during shutdown.
During this process the startup/shutdown mechanics were changed which introduced a lot of breaking changes.
Also included are improvements on the file, socket and scribe producers.
Write performance may show a minor increase for some producers.

This release contains breaking changes over version 0.3.x.
Custom producers and config files may have to be adjusted.

### Breaking changes with 0.4.0

* shared.RuntimeType renamed to TypeRegistry
* core.StreamTypes renamed to StreamRegistry
* ?ControlLoop callback parameters for command handling moved to callback members
* ?ControlLoop renamed to ?Loop, where ? can be a combination of Control (handling of control messages), Message (handling of messages) or Ticker (handling of regular callbacks)
* PluginControlStop is now splitted into PluginControlStopConsumer and PluginControlStopProducer to allow plugins that are producer and consumers.
* Producer.Enqueue now takes care of dropping messages and accepts a timeout overwrite value
* MessageBatch has been refactored to store messages instead of preformatted strings. This allows dropping messages from a batch.
* Message.Drop has been removed, Message.Route can be used instead
* The LoopBack consumer has been removed. Producers can now drop messages to any stream using DropToStream.
* Stream plugins are now allowed to only bind to one stream
* Renamed producer.HttpReq to producer.HTTPRequest
* Renamed format.StreamMod to format.StreamRoute
* For format.Envelope postfix and prefix configuration keys have been renamed to EnvelopePostifx and EnvelopePrefix
* Base64Encode and Base64Decode formatter parameters have been renamed to "Base64*"
* Removed the MessagesPerSecAvg metric
* Two functions were added to the MessageSource interface to allow blocked/active state query
* The low resolution timer has been removed

### Fixed with 0.4.0

* Messages stored in channels or MessageBatches can now be flushed properly during shutdown
* Several producers now properly block when their queue is full (messages could be lost before)
* Producer control commands now have priority over processing messages
* Switched to sarama trunk version to get the latest broker connection fixes
* Fixed various message loss scenarios in file, kafka and http request producer
* Kafka producer is now reconnecting upon every error (intermediate fix)
* StreamRoute formatter now properly works when the separator is a space
* File, Kafka and HTTPRequest plugins don't hava mandatory values anymore
* Socket consumer can now reopen a dropped connection
* Socket consumer can now change access rights on unix domain sockets
* Socket consumer now closes non-udp connections upon any error
* Socket consumer can now remove an existing UDS file with the same name if necessary
* Socket consumer now uses proper connection timeouts
* Socket consumer now sends special acks on error
* All net.Dial commands were replaced with net.DialTimeout
* The makfile now correctly includes the config folder
* Thie file producer now behaves correctly when directory creation fails
* Spinning loops are now more CPU friendly
* Plugins can now be addressed by longer paths, too, e.g. "contrib.company.sth"
* Log messages that appear during startup are now written to the set log producer, too
* Fixed a problem where control messages could cause shutdown to get stucked
* The Kafka producer has been rewritten for better error handling
* The scribe producer now dynamically modifies the batch size on error
* The metric server tries to reopen connection every 5 seconds
* Float metrics are now properly rounded
* Ticker functions are now restarted after the function is done, preventing double calls
* No empty messages will be sent during shutdown

### New with 0.4.0

* Added a new stream plugin to route messages to one or more other streams
* The file producer can now delete old files upon rotate (pruning)
* The file producer can now overwrite files and set file permissions
* Added metrics for dropped, discarded, filtered and unroutable messages
* Streams can now overwrite a producer's ChannelTimeoutMs setting (only for this stream)
* Producers are now shut down in order based on DropStream dependencies
* Messages now keep a one-step history of their StreamID
* Added format.StreamRevert to go back to the last used stream (e.g. after a drop)
* Added producer.Spooling that temporary stores messages to disk before trying them again (e.g. useful for disconnect scenarios)
* Added a new formatter to prepend stream names
* Added a new formatter to serialize messages
* Added a new formatter to convert collectd to InfluxDB (0.8.x and 0.9.x)
* It is now possible to add a custom string after the version number
* Plugins compiled from the contrib folder are now listed in the version string
* All producers can now define a filter applied before formatting
* Added unittests to check all bundled producer, consumer, format, filter and stream for interface compatibility
* Plugins can now be registered and queried by a string based ID via core.PluginRegistry
* Added producer for InfluxDB data (0.8.x and 0.9.x)
* Kafka, scribe and elastic search producers now have distinct metrics per topic/category/index
* Version number is now added to the metrics as in "MMmmpp" (M)ajor (m)inor (p)atch
