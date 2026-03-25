package dashboard

import (
	"testing"
	"time"
)

func TestSchemaCache_TTL(t *testing.T) {
	cache := NewSchemaCache(50*time.Millisecond, "")
	cache.Set("t1", SchemaV2)

	if v, ok := cache.Get("t1"); !ok || v != SchemaV2 {
		t.Fatalf("expected cache hit v2, got %v ok=%v", v, ok)
	}

	time.Sleep(60 * time.Millisecond)
	if _, ok := cache.Get("t1"); ok {
		t.Fatalf("expected cache miss after ttl")
	}
}

func TestSchemaCache_NoTTL(t *testing.T) {
	cache := NewSchemaCache(0, "")
	cache.Set("t1", SchemaV1)
	if v, ok := cache.Get("t1"); !ok || v != SchemaV1 {
		t.Fatalf("expected cache hit v1, got %v ok=%v", v, ok)
	}
}
