# 0.4.0

This release includes several reliability fixes that prevent messages from being lost during shutdown.
It also includes some minor improvements on the file, socket and scribe producers.
Write performance may show a minor increase for some producers.

This release contains breaking changes over version 0.3.x.
Custom producers and config files may have to be adjusted.

#### Breaking changes

 * Producers now have to implement a Close() method that is explicitly called during shutdown
 * The LoopBack consumer has been removed. Producers can now drop messages to any stream.
 * Producer.Enqueue now takes care of dropping messages and accepts a timeout overwrite value
 * MessageBatch has been refactored to store messages instead of preformatted strings. This allows dropping messages from a batch.
 * Stream plugins may now bind to exactly and only one stream
 * Message.Drop has been removed, Message.Route can now be used instead

#### Fixed

 * Messages stored in channels or MessageBatches can now be flushed properly during shutdown
 * Several producers now properly block when their queue is full (messages could be lost before)
 * Producer control commands now have priority over processing messages

#### New

 * Added a new stream plugin to route messages to one or more other streams
 * The file producer can now delete old files upon rotate (pruning)
 * Added metrics for dropped, discarded and unroutable messages
 * Streams can now overwrite a producer's ChannelTimeoutMs setting (only for this stream)
 * Producers are now shut down in order based on DropStream dependencies
