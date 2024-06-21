package enginenetx

// streamTacticsFromSlice streams tactics from a given slice.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func streamTacticsFromSlice(input []*httpsDialerTactic) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	go func() {
		defer close(output)
		for _, tx := range input {
			output <- tx
		}
	}()
	return output
}
