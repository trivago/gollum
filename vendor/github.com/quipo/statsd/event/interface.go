package event

// constant event type identifiers
const (
	EventIncr = iota
	EventTiming
	EventAbsolute
	EventTotal
	EventGauge
	EventGaugeDelta
	EventFGauge
	EventFGaugeDelta
	EventFAbsolute
	EventPrecisionTiming
)

// Event is an interface to a generic StatsD event, used by the buffered client collator
type Event interface {
	Stats() []string
	Type() int
	TypeString() string
	Payload() interface{}
	Update(e2 Event) error
	String() string
	Key() string
	SetKey(string)
}

// compile-time assertion to verify default events implement the Event interface
func _() {
	var _ Event = (*Absolute)(nil)        // assert *Absolute implements Event
	var _ Event = (*FAbsolute)(nil)       // assert *FAbsolute implements Event
	var _ Event = (*Gauge)(nil)           // assert *Gauge implements Event
	var _ Event = (*FGauge)(nil)          // assert *FGauge implements Event
	var _ Event = (*GaugeDelta)(nil)      // assert *GaugeDelta implements Event
	var _ Event = (*FGaugeDelta)(nil)     // assert *FGaugeDelta implements Event
	var _ Event = (*Increment)(nil)       // assert *Increment implements Event
	var _ Event = (*PrecisionTiming)(nil) // assert *PrecisionTiming implements Event
	var _ Event = (*Timing)(nil)          // assert *Timing implements Event
	var _ Event = (*Total)(nil)           // assert *Total implements Event
}
