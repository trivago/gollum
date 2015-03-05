package shared

var streamPaused = make(map[MessageStreamID]bool)

// PauseStream sets the pause state for a given stream
func PauseStream(streamID MessageStreamID) {
	streamPaused[streamID] = true
}

// ResumeStream unsets the pause state for a given stream
func ResumeStream(streamID MessageStreamID) {
	streamPaused[streamID] = false
}

// IsStreamPaused returns true if the given stream is marked as paused.
func IsStreamPaused(streamID MessageStreamID) bool {
	paused, exists := streamPaused[streamID]
	return exists && paused
}

// AllStreamsPaused returns true if all streams in the given array are paused.
// The wildcard stream is ignored here.
func AllStreamsPaused(streamIDs []MessageStreamID) bool {
	for _, streamID := range streamIDs {
		if streamID != WildcardStreamID && !IsStreamPaused(streamID) {
			return false
		}
	}
	return true
}

// AnyStreamPaused returns true if any stream in the given array is paused.
// The wildcard stream is ignored here.
func AnyStreamPaused(streamIDs []MessageStreamID) bool {
	for _, streamID := range streamIDs {
		if streamID != WildcardStreamID && IsStreamPaused(streamID) {
			return true
		}
	}
	return false
}
