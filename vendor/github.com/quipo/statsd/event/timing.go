package event

import "fmt"

// Timing keeps min/max/avg information about a timer over a certain interval
type Timing struct {
	Name  string
	Min   int64
	Max   int64
	Value int64
	Count int64
}

// NewTiming is a factory for a Timing event, setting the Count to 1 to prevent div_by_0 errors
func NewTiming(k string, delta int64) *Timing {
	return &Timing{Name: k, Min: delta, Max: delta, Value: delta, Count: 1}
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Timing) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	p := e2.Payload().(map[string]int64)
	e.Count += p["cnt"]
	e.Value += p["val"]
	e.Min = minInt64(e.Min, p["min"])
	e.Max = maxInt64(e.Max, p["max"])
	return nil
}

// Payload returns the aggregated value for this event
func (e Timing) Payload() interface{} {
	return map[string]int64{
		"min": e.Min,
		"max": e.Max,
		"val": e.Value,
		"cnt": e.Count,
	}
}

// Stats returns an array of StatsD events as they travel over UDP
func (e Timing) Stats() []string {
	return []string{
		fmt.Sprintf("%s.count:%d|c", e.Name, e.Count),
		fmt.Sprintf("%s.avg:%d|ms", e.Name, int64(e.Value/e.Count)), // make sure e.Count != 0
		fmt.Sprintf("%s.min:%d|ms", e.Name, e.Min),
		fmt.Sprintf("%s.max:%d|ms", e.Name, e.Max),
	}
}

// Key returns the name of this metric
func (e Timing) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Timing) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Timing) Type() int {
	return EventTiming
}

// TypeString returns a name for this type of metric
func (e Timing) TypeString() string {
	return "Timing"
}

// String returns a debug-friendly representation of this metric
func (e Timing) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %+v}", e.TypeString(), e.Name, e.Payload())
}

func minInt64(v1, v2 int64) int64 {
	if v1 <= v2 {
		return v1
	}
	return v2
}
func maxInt64(v1, v2 int64) int64 {
	if v1 >= v2 {
		return v1
	}
	return v2
}
