package testingx

import (
	"fmt"
	"net/http"
)

// GeoIPHandlerUbuntu is an [http.Handler] implementing Ubuntu's GeoIP lookup service.
type GeoIPHandlerUbuntu struct {
	// ProbeIP is the MANDATORY probe IP to return.
	ProbeIP string
}

var _ http.Handler = &GeoIPHandlerUbuntu{}

// ServeHTTP implements [http.Handler].
func (p *GeoIPHandlerUbuntu) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><Response><Ip>%s</Ip></Response>`,
		p.ProbeIP,
	)
	w.Header().Add("Content-Type", "text/xml")
	w.Write([]byte(resp))
}
