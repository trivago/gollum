package event

import "fmt"

// Total represents a metric that is continously increasing, e.g. read operations since boot
type Total struct {
	Name  string
	Value int64
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Total) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Value += e2.Payload().(int64)
	return nil
}

// Payload returns the aggregated value for this event
func (e Total) Payload() interface{} {
	return e.Value
}

// Stats returns an array of StatsD events as they travel over UDP
func (e Total) Stats() []string {
	return []string{fmt.Sprintf("%s:%d|t", e.Name, e.Value)}
}

// Key returns the name of this metric
func (e Total) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Total) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Total) Type() int {
	return EventTotal
}

// TypeString returns a name for this type of metric
func (e Total) TypeString() string {
	return "Total"
}

// String returns a debug-friendly representation of this metric
func (e Total) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %d}", e.TypeString(), e.Name, e.Value)
}
