package id

import (
	"context"
	"regexp"
	"testing"
)

var uuidV7 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestUUIDGeneratorProducesUniqueTimeOrderedV7IDs(t *testing.T) {
	previous := ""
	for range 100 {
		value, err := (UUIDGenerator{}).NewID(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !uuidV7.MatchString(value) {
			t.Fatalf("NewID() = %q, want a lowercase UUIDv7", value)
		}
		if value <= previous {
			t.Fatalf("NewID() = %q is not after %q; v7 ids must be unique and increasing", value, previous)
		}
		previous = value
	}
}
