package oonimkall

func (t *Task) IsRunning() bool {
	return t.isstopped.Load() == 0
}
