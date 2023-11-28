package webconnectivitylte

// httpRedirectIsRedirect returns true if the status code contains a redirect.
func httpRedirectIsRedirect(status int64) bool {
	switch status {
	case 301, 302, 307, 308:
		return true
	default:
		return false
	}
}
