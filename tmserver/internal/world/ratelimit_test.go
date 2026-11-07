package world

import (
	"testing"
	"time"
)

func TestTokenBucketDisabled(t *testing.T) {
	b := newTokenBucket(0, 0, time.Now())
	if b != nil {
		t.Fatal("non-positive refill should yield a disabled (nil) bucket")
	}
	// Um balde nil sempre permite.
	if !b.allow(time.Now()) {
		t.Fatal("nil bucket should allow")
	}
}

func TestTokenBucketBurstThenBlock(t *testing.T) {
	now := time.Unix(0, 0)
	b := newTokenBucket(10, 3, now) // rajada de 3, reenche a 10/s

	for i := 0; i < 3; i++ {
		if !b.allow(now) {
			t.Fatalf("burst token %d should be allowed", i)
		}
	}
	if b.allow(now) {
		t.Fatal("4th message in the same instant should be blocked")
	}

	// Depois de 100ms um token (10/s × 0.1s) é reposto.
	now = now.Add(100 * time.Millisecond)
	if !b.allow(now) {
		t.Fatal("a token should have refilled after 100ms")
	}
	if b.allow(now) {
		t.Fatal("only one token should have refilled")
	}
}

func TestTokenBucketCaps(t *testing.T) {
	now := time.Unix(0, 0)
	b := newTokenBucket(10, 3, now)
	// Ocioso por muito tempo: os tokens precisam parar em burst, não acumular sem limite.
	now = now.Add(10 * time.Second)
	allowed := 0
	for i := 0; i < 100; i++ {
		if b.allow(now) {
			allowed++
		}
	}
	if allowed != 3 {
		t.Fatalf("burst cap = %d, want 3", allowed)
	}
}
