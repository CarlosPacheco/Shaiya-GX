package npcquest

import "time"

// Deadline uses the earlier configured quest time limit.
func Deadline(maxMinutes, keepSeconds uint32) time.Duration {
	const maxDuration = time.Duration(1<<63 - 1)
	var limit time.Duration
	if maxMinutes != 0 {
		limit = min(time.Duration(maxMinutes), maxDuration/time.Minute) * time.Minute
	}
	if keep := time.Duration(keepSeconds) * time.Second; keep != 0 && (limit == 0 || keep < limit) {
		limit = keep
	}
	return limit
}

// PartyMeetsJobs counts the quest taker as part of a roster of at least two.
func PartyMeetsJobs(memberCount int, jobs, required [6]uint8) bool {
	if memberCount < 2 {
		return false
	}
	for i, count := range required {
		if jobs[i] < count {
			return false
		}
	}
	return true
}
