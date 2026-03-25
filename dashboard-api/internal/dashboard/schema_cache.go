package dashboard

import (
	"log"
	"sync"
	"time"
)

type SchemaCache struct {
	mu     sync.RWMutex
	items  map[string]cacheEntry
	ttl    time.Duration
	logger string
}

type cacheEntry struct {
	schema  SchemaType
	expires time.Time
}

func NewSchemaCache(ttl time.Duration, logPrefix string) *SchemaCache {
	return &SchemaCache{
		items:  make(map[string]cacheEntry),
		ttl:    ttl,
		logger: logPrefix,
	}
}

func (c *SchemaCache) Get(tenantID string) (SchemaType, bool) {
	if c == nil {
		return SchemaUnknown, false
	}
	c.mu.RLock()
	entry, ok := c.items[tenantID]
	c.mu.RUnlock()
	if !ok {
		log.Printf("%sschema cache miss tenant=%s", c.logger, tenantID)
		return SchemaUnknown, false
	}
	if c.ttl > 0 && time.Now().UTC().After(entry.expires) {
		log.Printf("%sschema cache expired tenant=%s", c.logger, tenantID)
		c.mu.Lock()
		delete(c.items, tenantID)
		c.mu.Unlock()
		return SchemaUnknown, false
	}
	log.Printf("%sschema cache hit tenant=%s schema=%s", c.logger, tenantID, entry.schema)
	return entry.schema, true
}

func (c *SchemaCache) Set(tenantID string, schema SchemaType) {
	if c == nil {
		return
	}
	exp := time.Time{}
	if c.ttl > 0 {
		exp = time.Now().UTC().Add(c.ttl)
	}
	c.mu.Lock()
	c.items[tenantID] = cacheEntry{schema: schema, expires: exp}
	c.mu.Unlock()
	log.Printf("%sschema cached tenant=%s schema=%s", c.logger, tenantID, schema)
}
