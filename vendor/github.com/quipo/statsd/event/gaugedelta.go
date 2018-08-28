package event

import "fmt"

// GaugeDelta - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again
type GaugeDelta struct {
	Name  string
	Value int64
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *GaugeDelta) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Value += e2.Payload().(int64)
	return nil
}

// Payload returns the aggregated value for this event
func (e GaugeDelta) Payload() interface{} {
	return e.Value
}

// Stats returns an array of StatsD events as they travel over UDP
func (e GaugeDelta) Stats() []string {
	return []string{fmt.Sprintf("%s:%+d|g", e.Name, e.Value)}
}

// Key returns the name of this metric
func (e GaugeDelta) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *GaugeDelta) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e GaugeDelta) Type() int {
	return EventGaugeDelta
}

// TypeString returns a name for this type of metric
func (e GaugeDelta) TypeString() string {
	return "GaugeDelta"
}

// String returns a debug-friendly representation of this metric
func (e GaugeDelta) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %d}", e.TypeString(), e.Name, e.Value)
}
