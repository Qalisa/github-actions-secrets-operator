package utils

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

func (r *SyncAttempts) HasFailed() bool       { return r.failed > 0 }
func (r *SyncAttempts) Total() int            { return r.total }
func (r *SyncAttempts) DoneOrDidSuccess() int { return r.notNeeded + r.successful }
