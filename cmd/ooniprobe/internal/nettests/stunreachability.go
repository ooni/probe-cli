package nettests

// STUNReachability nettest implementation.
type STUNReachability struct{}

// TODO: keep in sync with snowflake and ptx/snowflake.go
var stunReachabilityDefaultInput = []string{
	"stun://stun.voip.blackberry.com:3478",
	"stun://stun.altar.com.pl:3478",
	"stun://stun.antisip.com:3478",
	"stun://stun.bluesip.net:3478",
	"stun://stun.dus.net:3478",
	"stun://stun.epygi.com:3478",
	"stun://stun.sonetel.com:3478",
	"stun://stun.sonetel.net:3478",
	"stun://stun.stunprotocol.org:3478",
	"stun://stun.uls.co.za:3478",
	"stun://stun.voipgate.com:3478",
	"stun://stun.voys.nl:3478",
}

// Run starts the nettest.
func (n STUNReachability) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("stunreachability")
	if err != nil {
		return err
	}
	return ctl.Run(builder, stunReachabilityDefaultInput)
}
