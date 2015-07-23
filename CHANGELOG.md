# 0.4.0

This release includes several reliability fixes that prevent messages from being lost during shutdown.
It also includes some minor improvements on the file, socket and scribe producers.
Write performance may show a minor increase for some producers.

This release contains breaking changes over version 0.3.x.
Custom producers and config files may have to be adjusted.

#### Breaking changes

 * Producers now have to implement a Close() method that is explicitly called during shutdown
 * The LoopBack consumer has been removed. Producers can now drop messages to any stream using DropToStream.
 * Producer.Enqueue now takes care of dropping messages and accepts a timeout overwrite value
 * MessageBatch has been refactored to store messages instead of preformatted strings. This allows dropping messages from a batch.
 * Stream plugins may now bind to exactly and only one stream
 * Message.Drop has been removed, Message.Route can now be used instead
 * Renamed producer.HttpReq to producer.HTTPRequest
 * Renamed format.StreamMod to format.StreamRoute
 * ControlLoop* callback parameters for command handling moved to member functions
 * For format.Envelope postfix and prefix configuration keys have been renamed to EnvelopePostifx and EnvelopePrefix

#### Fixed

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
 * The makfile now correctly includes the config folder
 * Thie file producer now behaves correctly when directory creation fails

#### New

 * Added a new stream plugin to route messages to one or more other streams
 * The file producer can now delete old files upon rotate (pruning)
 * Added metrics for dropped, discarded and unroutable messages
 * Streams can now overwrite a producer's ChannelTimeoutMs setting (only for this stream)
 * Producers are now shut down in order based on DropStream dependencies
 * Messages now keep a one-step history of their StreamID
 * Added format.StreamRevert to go back to the last used stream (e.g. after a drop)
 * Added producer.Spooling that stores messages for a given time and resends it after this timeframe (usefull for drops)
 * New control message for consumers/producers before the shutdown starts
 * New formatter to prepend stream names
 * It is now possible to add a custom string after the version number
 * Kafka and scribe producers can now use a filter
