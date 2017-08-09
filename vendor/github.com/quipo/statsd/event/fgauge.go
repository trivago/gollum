package event

import "fmt"

// FGauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again
type FGauge struct {
	Name  string
	Value float64
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *FGauge) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.Value = e2.Payload().(float64)
	return nil
}

// Payload returns the aggregated value for this event
func (e FGauge) Payload() interface{} {
	return e.Value
}

// Stats returns an array of StatsD events as they travel over UDP
func (e FGauge) Stats() []string {
	if e.Value < 0 {
		// because a leading '+' or '-' in the value of a gauge denotes a delta, to send
		// a negative gauge value we first set the gauge absolutely to 0, then send the
		// negative value as a delta from 0 (that's just how the spec works :-)
		return []string{
			fmt.Sprintf("%s:%d|g", e.Name, 0),
			fmt.Sprintf("%s:%g|g", e.Name, e.Value),
		}
	}
	return []string{fmt.Sprintf("%s:%g|g", e.Name, e.Value)}
}

// Key returns the name of this metric
func (e FGauge) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *FGauge) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e FGauge) Type() int {
	return EventFGauge
}

// TypeString returns a name for this type of metric
func (e FGauge) TypeString() string {
	return "FGauge"
}

// String returns a debug-friendly representation of this metric
func (e FGauge) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %g}", e.TypeString(), e.Name, e.Value)
}
