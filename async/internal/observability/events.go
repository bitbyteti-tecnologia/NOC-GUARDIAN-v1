package observability

import "sync/atomic"

type EventCounters struct {
	created  uint64
	updated  uint64
	resolved uint64
	alerts   uint64
}

func NewEventCounters() *EventCounters {
	return &EventCounters{}
}

func (c *EventCounters) IncCreated()  { atomic.AddUint64(&c.created, 1) }
func (c *EventCounters) IncUpdated()  { atomic.AddUint64(&c.updated, 1) }
func (c *EventCounters) IncResolved() { atomic.AddUint64(&c.resolved, 1) }
func (c *EventCounters) IncAlerts()   { atomic.AddUint64(&c.alerts, 1) }

type EventSnapshot struct {
	Created  uint64
	Updated  uint64
	Resolved uint64
	Alerts   uint64
}

func (c *EventCounters) Snapshot() EventSnapshot {
	return EventSnapshot{
		Created:  atomic.LoadUint64(&c.created),
		Updated:  atomic.LoadUint64(&c.updated),
		Resolved: atomic.LoadUint64(&c.resolved),
		Alerts:   atomic.LoadUint64(&c.alerts),
	}
}
