// Package httpheader contains code to set common HTTP headers.
package httpheader

// UserAgent returns the User-Agent header used for measuring.
func UserAgent() string {
	// 13.2% as of Jan 24, 2021 according to https://techblog.willshouse.com/2012/01/03/most-common-user-agents/
	const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36"
	return ua
}

// CLIUserAgent returns the User-Agent used when we want to
// pretent to be a command line HTTP client.
func CLIUserAgent() string {
	// here we always put the latest version of cURL.
	return "curl/7.79.1"
}
