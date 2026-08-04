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

func TestDSTBoundariesRoundTripThroughUTC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		zone       string
		beforeUTC  time.Time
		afterUTC   time.Time
		beforeHour int
		afterHour  int
	}{
		{"London spring forward", "Europe/London", time.Date(2026, 3, 29, 0, 30, 0, 0, time.UTC), time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC), 0, 2},
		{"London fall back", "Europe/London", time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC), time.Date(2026, 10, 25, 1, 30, 0, 0, time.UTC), 1, 1},
		{"New York spring forward", "America/New_York", time.Date(2026, 3, 8, 6, 30, 0, 0, time.UTC), time.Date(2026, 3, 8, 7, 30, 0, 0, time.UTC), 1, 3},
		{"New York fall back", "America/New_York", time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC), time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC), 1, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			location, err := time.LoadLocation(test.zone)
			if err != nil {
				t.Fatalf("LoadLocation: %v", err)
			}
			beforeLocal := test.beforeUTC.In(location)
			afterLocal := test.afterUTC.In(location)
			if beforeLocal.Hour() != test.beforeHour || afterLocal.Hour() != test.afterHour {
				t.Fatalf("local hours = %d -> %d, want %d -> %d", beforeLocal.Hour(), afterLocal.Hour(), test.beforeHour, test.afterHour)
			}
			if !beforeLocal.UTC().Equal(test.beforeUTC) || !afterLocal.UTC().Equal(test.afterUTC) {
				t.Fatal("local conversion did not round-trip through UTC")
			}
			if got := test.beforeUTC.Add(time.Hour); !got.Equal(test.afterUTC) {
				t.Fatalf("UTC arithmetic = %v, want %v", got, test.afterUTC)
			}
		})
	}
}

func TestEmbeddedTZDataLoadsNonUTCZone(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("Pacific/Chatham")
	if err != nil {
		t.Fatalf("load IANA zone from embedded tzdata: %v", err)
	}
	_, offset := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC).In(location).Zone()
	if offset != 12*60*60+45*60 {
		t.Fatalf("Pacific/Chatham offset = %d, want +12:45", offset)
	}
}
