package minipipeline

func utilsStringPointerToString(failure *string) (out string) {
	if failure != nil {
		out = *failure
	}
	return
}
