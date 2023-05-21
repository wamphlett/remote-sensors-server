package manager

import (
	"encoding/json"
	"math"
	"sync"
	"time"
)

type average struct {
	sum   float64
	count int
}

func (a *average) result() float64 {
	if a.sum == 0 || a.count == 0 {
		return 0
	}

	output := math.Pow(10, float64(2))
	result := a.sum / float64(a.count)
	return float64(math.Round(result*output)) / output

}

// AggregatedMetric provides a way to record metric values and will return averages
// as well as the last value
type AggregatedMetric struct {
	sync.Mutex
	values map[time.Time]float64
	stop   chan bool
}

// NewAggregatedMetric returns a new AggregatedMetric and starts the periodic
// Clean function
func NewAggregatedMetric() *AggregatedMetric {
	a := &AggregatedMetric{
		values: make(map[time.Time]float64),
		stop:   make(chan bool),
	}

	go a.Clean()

	return a
}

// Stop will stop the Clean function
func (a *AggregatedMetric) Stop() {
	a.stop <- true
}

// Update will add a new value to the aggregation values
func (a *AggregatedMetric) Update(v float64) {
	a.Lock()
	defer a.Unlock()
	a.values[time.Now()] = v
}

// Clean will periodically remove old values
func (a *AggregatedMetric) Clean() {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-a.stop:
			return
		case <-ticker.C:
			a.Lock()
			for t, _ := range a.values {
				if time.Since(t) > time.Second*30 {
					delete(a.values, t)
				}
			}
			a.Unlock()
		}
	}
}

// MarshalJSON provides custom JSON formatting
func (a *AggregatedMetric) MarshalJSON() ([]byte, error) {
	a.Lock()
	defer a.Unlock()

	avg10s := &average{}
	avg30s := &average{}

	var mostRecentTime time.Time
	for t, v := range a.values {
		if t.After(mostRecentTime) {
			mostRecentTime = t
		}

		if time.Since(t) < 10*time.Second {
			avg10s.sum += v
			avg10s.count++
		}

		if time.Since(t) < 30*time.Second {
			avg30s.sum += v
			avg30s.count++
		}
	}

	return json.Marshal(&struct {
		Last   float64 `json:"last"`
		Avg10s float64 `json:"10s"`
		Avg30s float64 `json:"30s"`
	}{
		Last:   a.values[mostRecentTime],
		Avg10s: avg10s.result(),
		Avg30s: avg30s.result(),
	})
}
