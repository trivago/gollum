package shared

var messageFeedbackQueue chan Message

// EnableFeedbackQueue initializes the feedback queue with a given size.
// This function will not change the feedback queue once it has been initialized.
func EnableFeedbackQueue(size int) {
	if messageFeedbackQueue == nil {
		messageFeedbackQueue = make(chan Message, size)
	}
}

// GetFeedbackQueue returns read access to the feedback queue
func GetFeedbackQueue() <-chan Message {
	return messageFeedbackQueue
}

// FeedbackMessage sends a message to the feedback queue for the given stream.
// If the queue is not initialized the message is dropped.
func FeedbackMessage(msg Message, streamID MessageStreamID) {
	if messageFeedbackQueue != nil {
		msg.CurrentStream = streamID
		messageFeedbackQueue <- msg
	}
}

// DropMessage sends a message to the _DROPPED_ feedback stream.
func DropMessage(msg Message) {
	FeedbackMessage(msg, DroppedStreamID)
}
