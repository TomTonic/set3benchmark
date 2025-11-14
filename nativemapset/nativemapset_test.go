package nativemapset

import (
	"testing"
)

func TestEmptyNativeMapSetWithCapacity(t *testing.T) {
	set := EmptyNativeMapSetWithCapacity[int](0)
	if set == nil {
		t.Error("Expected non-nil set")
	}
	if set.Count() != uint32(0) {
		t.Errorf("Expected count to be 0, got %d", set.Count())
	}
}

func TestNativeMapSetAddAndContains(t *testing.T) {
	set := EmptyNativeMapSetWithCapacity[int](0)
	set.Add(1)
	set.Add(2)
	set.Add(2)
	set.Add(3)

	if !set.Contains(1) {
		t.Error("Expected set to contain 1")
	}
	if !set.Contains(2) {
		t.Error("Expected set to contain 2")
	}
	if !set.Contains(3) {
		t.Error("Expected set to contain 3")
	}
	if set.Contains(4) {
		t.Error("Expected set not to contain 4")
	}
}

func TestNativeMapSetCount(t *testing.T) {
	set := EmptyNativeMapSetWithCapacity[int](0)
	set.Add(1)
	set.Add(2)
	set.Add(3)

	if set.Count() != 3 {
		t.Errorf("Expected count to be 3, got %d", set.Count())
	}

	set.Add(3) // Adding duplicate
	if set.Count() != 3 {
		t.Errorf("Expected count to be 3 after adding duplicate, got %d", set.Count())
	}
}

func TestNativeMapSetClear(t *testing.T) {
	set := EmptyNativeMapSetWithCapacity[int](0)
	set.Add(1)
	set.Add(2)
	set.Add(3)

	if set.Count() != 3 {
		t.Errorf("Expected count to be 3, got %d", set.Count())
	}

	set.Clear()

	if set.Count() != 0 {
		t.Errorf("Expected count to be 0 after clearing, got %d", set.Count())
	}
}
