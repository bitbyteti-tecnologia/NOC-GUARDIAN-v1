package observability

import (
	"sync/atomic"
	"time"
)

type Counters struct {
	processed uint64
	failed    uint64
	retried   uint64
	dead      uint64
	durations uint64
	totalMS   uint64
}

func NewCounters() *Counters {
	return &Counters{}
}

func (c *Counters) IncProcessed() { atomic.AddUint64(&c.processed, 1) }
func (c *Counters) IncFailed()    { atomic.AddUint64(&c.failed, 1) }
func (c *Counters) IncRetried()   { atomic.AddUint64(&c.retried, 1) }
func (c *Counters) IncDead()      { atomic.AddUint64(&c.dead, 1) }

func (c *Counters) ObserveDuration(d time.Duration) {
	atomic.AddUint64(&c.durations, 1)
	atomic.AddUint64(&c.totalMS, uint64(d.Milliseconds()))
}

type Snapshot struct {
	Processed uint64
	Failed    uint64
	Retried   uint64
	Dead      uint64
	AvgMS     float64
}

func (c *Counters) Snapshot() Snapshot {
	processed := atomic.LoadUint64(&c.processed)
	failed := atomic.LoadUint64(&c.failed)
	retried := atomic.LoadUint64(&c.retried)
	dead := atomic.LoadUint64(&c.dead)
	durations := atomic.LoadUint64(&c.durations)
	totalMS := atomic.LoadUint64(&c.totalMS)

	avg := 0.0
	if durations > 0 {
		avg = float64(totalMS) / float64(durations)
	}

	return Snapshot{
		Processed: processed,
		Failed:    failed,
		Retried:   retried,
		Dead:      dead,
		AvgMS:     avg,
	}
}
