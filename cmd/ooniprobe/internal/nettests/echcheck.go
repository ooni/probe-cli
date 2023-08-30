package nettests

// ECHCheck nettest implementation.
type ECHCheck struct{}

// we use single input as an experimental setup to collect the first baseline measurements
var exampleInput = []string{"https://www.example.com"}

// Run starts the nettest.
func (n ECHCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("echcheck")
	if err != nil {
		return err
	}
	return ctl.Run(builder, exampleInput)
}
