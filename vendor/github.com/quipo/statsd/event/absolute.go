package event

import "fmt"

// Absolute is a metric that is not averaged/aggregated.
// We keep each value distinct and then we flush them all individually.
type Absolute struct {
	Name   string
	Values []int64
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Absolute) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Values = append(e.Values, e2.Payload().([]int64)...)
	return nil
}

// Payload returns the aggregated value for this event
func (e Absolute) Payload() interface{} {
	return e.Values
}

// Stats returns an array of StatsD events as they travel over UDP
func (e Absolute) Stats() []string {
	ret := make([]string, 0, len(e.Values))
	for _, v := range e.Values {
		ret = append(ret, fmt.Sprintf("%s:%d|a", e.Name, v))
	}
	return ret
}

// Key returns the name of this metric
func (e Absolute) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Absolute) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Absolute) Type() int {
	return EventAbsolute
}

// TypeString returns a name for this type of metric
func (e Absolute) TypeString() string {
	return "Absolute"
}

// String returns a debug-friendly representation of this metric
func (e Absolute) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Values: %v}", e.TypeString(), e.Name, e.Values)
}
