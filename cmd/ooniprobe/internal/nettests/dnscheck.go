package nettests

// DNSCheck nettest implementation.
type DNSCheck struct{}

// TODO(https://github.com/ooni/probe/issues/1390): we need to
// implement serving DNSCheck targets from the API
var dnsCheckDefaultInput = mustStringListToModelURLInfo([]string{
	"https://dns.google/dns-query",
	"https://8.8.8.8/dns-query",
	"dot://8.8.8.8:853/",
	"dot://8.8.4.4:853/",
	"https://8.8.4.4/dns-query",
	"https://cloudflare-dns.com/dns-query",
	"https://1.1.1.1/dns-query",
	"https://1.0.0.1/dns-query",
	"dot://1.1.1.1:853/",
	"dot://1.0.0.1:853/",
	"https://dns.quad9.net/dns-query",
	"https://9.9.9.9/dns-query",
	"dot://9.9.9.9:853/",
	"dot://dns.quad9.net/",
})

// Run starts the nettest.
func (n DNSCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("dnscheck")
	if err != nil {
		return err
	}
	input, err := ctl.BuildAndSetInputIdxMap(ctl.Probe.DB(), dnsCheckDefaultInput)
	if err != nil {
		return err
	}
	return ctl.Run(builder, input)
}
