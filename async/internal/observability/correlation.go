package observability

import "sync/atomic"

type CorrelationCounters struct {
	created  uint64
	updated  uint64
	resolved uint64
}

func NewCorrelationCounters() *CorrelationCounters {
	return &CorrelationCounters{}
}

func (c *CorrelationCounters) IncCreated()  { atomic.AddUint64(&c.created, 1) }
func (c *CorrelationCounters) IncUpdated()  { atomic.AddUint64(&c.updated, 1) }
func (c *CorrelationCounters) IncResolved() { atomic.AddUint64(&c.resolved, 1) }

type CorrelationSnapshot struct {
	Created  uint64
	Updated  uint64
	Resolved uint64
}

func (c *CorrelationCounters) Snapshot() CorrelationSnapshot {
	return CorrelationSnapshot{
		Created:  atomic.LoadUint64(&c.created),
		Updated:  atomic.LoadUint64(&c.updated),
		Resolved: atomic.LoadUint64(&c.resolved),
	}
}
