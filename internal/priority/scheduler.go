package priority

// Scheduler applies HIGH/MEDIUM fairness counters when selecting the next lane.
type Scheduler struct {
	highEvery   int
	mediumEvery int

	highSinceMedium int
	mediumSinceLow  int
	highSinceLow    int
}

// NewScheduler builds a fairness scheduler. highEvery and mediumEvery must be >= 1.
func NewScheduler(highEvery, mediumEvery int) *Scheduler {
	if highEvery < 1 {
		highEvery = 1
	}
	if mediumEvery < 1 {
		mediumEvery = 1
	}
	return &Scheduler{
		highEvery:   highEvery,
		mediumEvery: mediumEvery,
	}
}

// Availability reports whether each lane currently has work.
type Availability struct {
	Critical bool
	High     bool
	Medium   bool
	Low      bool
}

func (s *Scheduler) mediumDue() bool {
	return s.highSinceMedium >= s.highEvery
}

func (s *Scheduler) lowDueFromMedium() bool {
	return s.mediumSinceLow >= s.mediumEvery
}

func (s *Scheduler) lowDueFromHigh() bool {
	return s.highSinceLow >= s.highEvery*s.mediumEvery
}

func (s *Scheduler) lowDue() bool {
	return s.lowDueFromMedium() || s.lowDueFromHigh()
}

// Pick selects the next lane given current availability. Returns None when idle.
// A due LOW is preferred over a due MEDIUM so that HIGH*MEDIUM credits can
// unlock LOW even when highSinceMedium has also reached its threshold.
func (s *Scheduler) Pick(avail Availability) Level {
	if avail.Critical {
		return Critical
	}
	if avail.Low && s.lowDue() {
		return Low
	}
	if avail.Medium && s.mediumDue() {
		return Medium
	}
	if avail.High {
		return High
	}
	if avail.Medium {
		return Medium
	}
	if avail.Low {
		return Low
	}
	return None
}

// Record updates fairness counters after a message from level was processed.
func (s *Scheduler) Record(level Level) {
	switch level {
	case Critical:
		return
	case High:
		s.highSinceMedium++
		s.highSinceLow++
	case Medium:
		if s.mediumDue() {
			s.highSinceMedium = 0
		}
		s.mediumSinceLow++
	case Low:
		if s.lowDueFromMedium() {
			s.mediumSinceLow = 0
		}
		if s.lowDueFromHigh() {
			s.highSinceLow = 0
		}
	}
}
