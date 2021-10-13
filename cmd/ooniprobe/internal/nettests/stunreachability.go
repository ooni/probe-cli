package nettests

// STUNReachability nettest implementation.
type STUNReachability struct{}

// TODO: keep in sync with snowflake and ptx/snowflake.go
var stunReachabilityDefaultInput = []string{
	"stun.voip.blackberry.com:3478",
	"stun.altar.com.pl:3478",
	"stun.antisip.com:3478",
	"stun.bluesip.net:3478",
	"stun.dus.net:3478",
	"stun.epygi.com:3478",
	"stun.sonetel.com:3478",
	"stun.sonetel.net:3478",
	"stun.stunprotocol.org:3478",
	"stun.uls.co.za:3478",
	"stun.voipgate.com:3478",
	"stun.voys.nl:3478",
}

// Run starts the nettest.
func (n STUNReachability) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("stunreachability")
	if err != nil {
		return err
	}
	return ctl.Run(builder, stunReachabilityDefaultInput)
}
