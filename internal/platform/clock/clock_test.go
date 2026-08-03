package clock

import (
	"testing"
	"time"
)

func TestToContractUTCMicrosecond(t *testing.T) {
	t.Parallel()

	in := time.Date(2026, 8, 3, 12, 0, 0, 491822137, time.FixedZone("CET", 3600))
	got := toContract(in)

	if got.Location() != time.UTC {
		t.Fatalf("location: got %v want UTC", got.Location())
	}
	if got.Nanosecond()%1000 != 0 {
		t.Fatalf("nanos %d not truncated to microseconds", got.Nanosecond())
	}
	want := time.Date(2026, 8, 3, 11, 0, 0, 491822000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSystemNowUTCMicrosecond(t *testing.T) {
	t.Parallel()

	now := System{}.Now()
	if now.Location() != time.UTC {
		t.Fatalf("location: got %v want UTC", now.Location())
	}
	if now.Nanosecond()%1000 != 0 {
		t.Fatalf("nanos %d not truncated to microseconds", now.Nanosecond())
	}
}

func TestFixedNowUpholdsContract(t *testing.T) {
	t.Parallel()

	instant := time.Date(2026, 8, 3, 12, 0, 0, 491822137, time.FixedZone("CET", 3600))
	got := Fixed{Instant: instant}.Now()
	want := toContract(instant)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got.Location() != time.UTC {
		t.Fatalf("location: got %v want UTC", got.Location())
	}
	if got.Nanosecond()%1000 != 0 {
		t.Fatalf("nanos %d not truncated to microseconds", got.Nanosecond())
	}
}
