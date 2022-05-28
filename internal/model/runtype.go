package model

// RunType describes the type of a ooniprobe run.
type RunType string

const (
	// RunTypeManual indicates that the user manually run `ooniprobe run`. Command
	// line tools such as miniooni should always use this run type.
	RunTypeManual = RunType("manual")

	// RunTypeTimed indicates that the user run `ooniprobe run unattended`, which
	// is the correct way to run ooniprobe from scripts and cronjobs.
	RunTypeTimed = RunType("timed")
)
