package measurexlite

//
// Code to process web results (e.g., from web connectivity)
//

import "regexp"

// webTitleRegexp is the regexp to extract the title
//
// MK used {1,128} but we're making it larger here to get longer titles
// e.g. <http://www.isa.gov.il/Pages/default.aspx>'s one
var webTitleRegexp = regexp.MustCompile(`(?i)<title>([^<]{1,512})</title>`)

// WebGetTitle returns the title or an empty string.
func WebGetTitle(measurementBody string) string {
	v := webTitleRegexp.FindStringSubmatch(measurementBody)
	if len(v) < 2 {
		return ""
	}
	return v[1]
}
