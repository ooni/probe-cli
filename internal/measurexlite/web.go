package measurexlite

//
// Code to process web results (e.g., from web connectivity)
//

import "regexp"

// WebGetTitle returns the title or an empty string.
func WebGetTitle(measurementBody string) string {
	// MK used {1,128} but we're making it larger here to get longer titles
	// e.g. <http://www.isa.gov.il/Pages/default.aspx>'s one
	re := regexp.MustCompile(`(?i)<title>([^<]{1,512})</title>`)
	v := re.FindStringSubmatch(measurementBody)
	if len(v) < 2 {
		return ""
	}
	return v[1]
}
