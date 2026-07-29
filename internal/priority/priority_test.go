package priority_test

import (
	"testing"

	"github.com/mywebsite/construction-ai-gateway/internal/priority"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want priority.Level
	}{
		{in: "CRITICAL", want: priority.Critical},
		{in: "critical", want: priority.Critical},
		{in: " HIGH ", want: priority.High},
		{in: "medium", want: priority.Medium},
		{in: "LOW", want: priority.Low},
		{in: "", want: priority.Low},
		{in: "urgent", want: priority.Low},
	}

	for _, tt := range tests {
		if got := priority.Parse(tt.in); got != tt.want {
			t.Fatalf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSchedulerCriticalFirst(t *testing.T) {
	s := priority.NewScheduler(3, 3)
	got := s.Pick(priority.Availability{Critical: true, High: true, Medium: true, Low: true})
	if got != priority.Critical {
		t.Fatalf("Pick() = %q, want CRITICAL", got)
	}
	s.Record(priority.Critical)
	if got := s.Pick(priority.Availability{High: true}); got != priority.High {
		t.Fatalf("after critical, Pick() = %q, want HIGH", got)
	}
}

func TestSchedulerMediumAfterHighCount(t *testing.T) {
	s := priority.NewScheduler(3, 3)
	avail := priority.Availability{High: true, Medium: true}

	for i := 0; i < 3; i++ {
		got := s.Pick(avail)
		if got != priority.High {
			t.Fatalf("step %d: Pick() = %q, want HIGH", i, got)
		}
		s.Record(priority.High)
	}

	got := s.Pick(avail)
	if got != priority.Medium {
		t.Fatalf("after 3 HIGH: Pick() = %q, want MEDIUM", got)
	}
	s.Record(priority.Medium)

	got = s.Pick(avail)
	if got != priority.High {
		t.Fatalf("after medium slot: Pick() = %q, want HIGH", got)
	}
}

func TestSchedulerLowAfterNineHighAndTwoMedium(t *testing.T) {
	s := priority.NewScheduler(3, 3)

	// Process 9 HIGH with fairness MEDIUM inserts every 3 HIGH.
	// Pattern: H H H M  H H H M  H H H  → 9 HIGH + 2 MEDIUM, then LOW due from high.
	sequence := []priority.Level{
		priority.High, priority.High, priority.High, priority.Medium,
		priority.High, priority.High, priority.High, priority.Medium,
		priority.High, priority.High, priority.High,
	}
	for i, want := range sequence {
		got := s.Pick(priority.Availability{High: true, Medium: true, Low: true})
		if got != want {
			t.Fatalf("step %d: Pick() = %q, want %q", i, got, want)
		}
		s.Record(got)
	}

	got := s.Pick(priority.Availability{High: true, Medium: true, Low: true})
	if got != priority.Low {
		t.Fatalf("after 9 HIGH + 2 MEDIUM: Pick() = %q, want LOW", got)
	}
	s.Record(priority.Low)

	// medium is still due (highSinceMedium == 3); processing it brings mediumSinceLow to 3 → another LOW.
	got = s.Pick(priority.Availability{High: true, Medium: true, Low: true})
	if got != priority.Medium {
		t.Fatalf("after low from high: Pick() = %q, want MEDIUM", got)
	}
	s.Record(priority.Medium)

	got = s.Pick(priority.Availability{High: true, Medium: true, Low: true})
	if got != priority.Low {
		t.Fatalf("after +1 MEDIUM: Pick() = %q, want LOW", got)
	}
}

func TestSchedulerLowAfterThreeMedium(t *testing.T) {
	s := priority.NewScheduler(3, 3)
	for i := 0; i < 3; i++ {
		got := s.Pick(priority.Availability{Medium: true, Low: true})
		if got != priority.Medium {
			t.Fatalf("step %d: Pick() = %q, want MEDIUM", i, got)
		}
		s.Record(priority.Medium)
	}
	got := s.Pick(priority.Availability{Medium: true, Low: true})
	if got != priority.Low {
		t.Fatalf("after 3 MEDIUM: Pick() = %q, want LOW", got)
	}
}

func TestSchedulerFallsBackWhenDueLaneEmpty(t *testing.T) {
	s := priority.NewScheduler(3, 3)
	for i := 0; i < 3; i++ {
		s.Record(priority.High)
	}
	// Medium is due but empty → continue with HIGH.
	got := s.Pick(priority.Availability{High: true, Low: true})
	if got != priority.High {
		t.Fatalf("medium due but empty: Pick() = %q, want HIGH", got)
	}
}
