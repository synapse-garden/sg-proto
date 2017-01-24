package task

// ByOldest is a sort.Interface implementation for a slice of *Task to
// be sorted by due date: from oldest to newest, followed by tasks
// without a due date.
type ByOldest []*Task

func (t ByOldest) Len() int      { return len(t) }
func (t ByOldest) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func (t ByOldest) Less(i, j int) bool {
	tI, tJ := t[i].Due, t[j].Due
	switch {
	case tI == nil && tJ == nil:
		return false
	case tI == nil:
		return false
	case tJ == nil:
		return true
	default:
		return tI.Before(*tJ)
	}
}
