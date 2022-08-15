package webconnectivity

const (
	// analysisBlockingDNS indicates there's blocking at the DNS level.
	analysisBlockingDNS = 1 << iota

	// analysisBlockingTCPIP indicates there's blocking at the TCP/IP level.
	analysisBlockingTCPIP

	// analysisBlockingHTTP indicates there's blocking at the HTTP level.
	analysisBlockingHTTP
)

// analysisToplevel is the toplevel function that analyses the results
// of the experiment once all network tasks have completed.
func (tk *TestKeys) analysisToplevel() {
	tk.analysisDNSToplevel()
}
