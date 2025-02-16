package utils

import (
	"fmt"
	"strings"
)

// Register all attempts of sync against Github API in a single sync run on a repo
type SyncAttempts struct {
	notNeeded  int
	successful int
	failed     int
	total      int
}

func (r *SyncAttempts) BumpTotal()      { r.total++ }
func (r *SyncAttempts) BumpFailed()     { r.failed++ }
func (r *SyncAttempts) BumpNotNeeded()  { r.notNeeded++ }
func (r *SyncAttempts) BumpSuccessful() { r.successful++ }

// if failed to sync a property, even once
func (r *SyncAttempts) HasEverFailed() bool { return r.failed > 0 }

func (r *SyncAttempts) Total() int { return r.total }

func (r *SyncAttempts) SuccessfulWithSkipped() int {
	return r.notNeeded + r.successful
}

//
//
//

type SyncAttemptsByType = map[GithubActionSecVarType]*SyncAttempts

func SyncAttempts_AnyHasFailed(attempsByTypes SyncAttemptsByType) bool {
	for _, attemps := range attempsByTypes {
		if attemps.HasEverFailed() {
			return true
		}
	}
	return false
}

func SyncAttempts_ProduceStats(attempsByTypes SyncAttemptsByType) string {
	//
	statsByType := []string{}

	//
	for attemptType, attemps := range attempsByTypes {
		statStr := fmt.Sprintf(
			"%d/%d %s",
			attemps.SuccessfulWithSkipped(), attemps.Total(),
			attemptType.String(),
		)
		statsByType = append(statsByType, statStr)
	}

	//
	return fmt.Sprintf("(Synced: %s)", strings.Join(statsByType, " | "))
}
