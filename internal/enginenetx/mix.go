package enginenetx

// mixSequentially mixes entries from primary followed by entries from fallback.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func mixSequentially(primary, fallback <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		defer close(output)
		for tx := range primary {
			output <- tx
		}
		for tx := range fallback {
			output <- tx
		}
	}()
	return output
}
