package netemx

import "net/http"

type GeoIPLookup struct{}

func (p *GeoIPLookup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := `<?xml version="1.0" encoding="UTF-8"?><Response><Ip>89.0.2.153</Ip><Status>OK</Status><CountryCode>DE</CountryCode><CountryCode3>DEU</CountryCode3><CountryName>Germany</CountryName><RegionCode>07</RegionCode><RegionName>Nordrhein-Westfalen</RegionName><City>Aachen</City><ZipPostalCode>52074</ZipPostalCode><Latitude>50.7479</Latitude><Longitude>6.0485</Longitude><AreaCode>0</AreaCode><TimeZone>Europe/Berlin</TimeZone></Response>`

	w.Header().Add("Content-Type", "text/xml")
	w.Write([]byte(resp))
}
