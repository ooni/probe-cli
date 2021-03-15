package nettests

// STUNReachability nettest implementation.
type STUNReachability struct{}

// Run starts the nettest.
func (n STUNReachability) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("stunreachability")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
