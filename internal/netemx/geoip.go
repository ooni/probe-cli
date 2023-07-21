package netemx

import "net/http"

type GeoIPLookup struct{}

func (p *GeoIPLookup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := `<?xml version="1.0" encoding="UTF-8"?><Response><Ip>99.83.231.61</Ip><Status>OK</Status><CountryCode>US</CountryCode><CountryCode3>USA</CountryCode3><CountryName>United States of America</CountryName><RegionName>Washington</RegionName><City>Seattle</City><ZipPostalCode>98108</ZipPostalCode><Latitude>47.5413</Latitude><Longitude>-122.3129</Longitude><TimeZone>America/Los_Angeles</TimeZone></Response>`

	w.Header().Add("Content-Type", "text/xml")
	w.Write([]byte(resp))
}
