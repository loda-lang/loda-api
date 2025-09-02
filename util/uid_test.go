package util

import (
	"testing"
)

func TestNewUID(t *testing.T) {
	u, err := NewUID('A', 123456)
	if err != nil {
		t.Fatalf("NewUID failed: %v", err)
	}
	if u.Domain() != 'A' || u.Number() != 123456 {
		t.Errorf("NewUID: got %c %d, want %c %d", u.Domain(), u.Number(), 'A', 123456)
	}

	_, err = NewUID('a', 123456)
	if err == nil {
		t.Errorf("NewUID: expected error for invalid domain")
	}
	_, err = NewUID('A', -1)
	if err == nil {
		t.Errorf("NewUID: expected error for negative id")
	}
	_, err = NewUID('A', 1000000)
	if err == nil {
		t.Errorf("NewUID: expected error for id out of range")
	}
}

func TestNewUIDFromString(t *testing.T) {
	u, err := NewUIDFromString("A123456")
	if err != nil {
		t.Fatalf("NewUIDFromString failed: %v", err)
	}
	if u.Domain() != 'A' || u.Number() != 123456 {
		t.Errorf("NewUIDFromString: got %c %d, want %c %d", u.Domain(), u.Number(), 'A', 123456)
	}

	_, err = NewUIDFromString("")
	if err == nil {
		t.Errorf("NewUIDFromString: expected error for empty string")
	}
	_, err = NewUIDFromString("A")
	if err == nil {
		t.Errorf("NewUIDFromString: expected error for short string")
	}
	_, err = NewUIDFromString("Aabcdef")
	if err == nil {
		t.Errorf("NewUIDFromString: expected error for non-numeric id")
	}
}

func TestUIDString(t *testing.T) {
	u, _ := NewUID('B', 42)
	got := u.String()
	want := "B000042"
	if got != want {
		t.Errorf("UID.String: got %q, want %q", got, want)
	}
}

func TestUIDEquals(t *testing.T) {
	u1, _ := NewUID('A', 1)
	u2, _ := NewUID('A', 1)
	u3, _ := NewUID('A', 2)
	u4, _ := NewUID('B', 1)
	if !u1.Equals(u2) {
		t.Errorf("UID.Equals: expected true")
	}
	if u1.Equals(u3) {
		t.Errorf("UID.Equals: expected false")
	}
	if u1.Equals(u4) {
		t.Errorf("UID.Equals: expected false")
	}
}

func TestUIDComparison(t *testing.T) {
	u1, _ := NewUID('A', 1)
	u2, _ := NewUID('A', 2)
	u3, _ := NewUID('B', 1)
	if !u1.IsLessThan(u2) {
		t.Errorf("UID.IsLessThan: expected true")
	}
	if u2.IsLessThan(u1) {
		t.Errorf("UID.IsLessThan: expected false")
	}
	if !u1.IsLessThan(u3) {
		t.Errorf("UID.IsLessThan: expected true")
	}
	if u3.IsLessThan(u1) {
		t.Errorf("UID.IsLessThan: expected false")
	}
	if !u3.IsGreaterThan(u1) {
		t.Errorf("UID.IsGreaterThan: expected true")
	}
}

func TestUIDIsZero(t *testing.T) {
	u, _ := NewUID('A', 0)
	if !u.IsZero() {
		t.Errorf("UID.IsZero: expected true")
	}
	u2, _ := NewUID('B', 0)
	if u2.IsZero() {
		t.Errorf("UID.IsZero: expected false")
	}
}
