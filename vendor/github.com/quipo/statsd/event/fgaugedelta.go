package event

import "fmt"

// FGaugeDelta - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again
type FGaugeDelta struct {
	Name  string
	Value float64
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *FGaugeDelta) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Value += e2.Payload().(float64)
	return nil
}

// Payload returns the aggregated value for this event
func (e FGaugeDelta) Payload() interface{} {
	return e.Value
}

// Stats returns an array of StatsD events as they travel over UDP
func (e FGaugeDelta) Stats() []string {
	return []string{fmt.Sprintf("%s:%+g|g", e.Name, e.Value)}
}

// Key returns the name of this metric
func (e FGaugeDelta) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *FGaugeDelta) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e FGaugeDelta) Type() int {
	return EventFGaugeDelta
}

// TypeString returns a name for this type of metric
func (e FGaugeDelta) TypeString() string {
	return "FGaugeDelta"
}

// String returns a debug-friendly representation of this metric
func (e FGaugeDelta) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %g}", e.TypeString(), e.Name, e.Value)
}
