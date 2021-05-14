package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
)

func main() {
	address := flag.String("address", "127.0.0.1:8002", "Set listening address")
	user := flag.String("username", "", "Optional authentication username")
	password := flag.String("password", "", "Optional authentication password")
	flag.Parse()
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	if *user != "" && *password != "" {
		proxy.OnRequest().HandleConnect(
			auth.BasicConnect("realm.ooni.org", func(u, p string) bool {
				return u == *user && p == *password
			}))
	}
	log.Fatal(http.ListenAndServe(*address, proxy))
}
