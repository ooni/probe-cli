package main

import (
	"flag"

	"github.com/armon/go-socks5"
)

func makeCredentials(user, password string) socks5.StaticCredentials {
	if user == "" && password == "" {
		return nil
	}
	return socks5.StaticCredentials{user: password}
}

func main() {
	address := flag.String("address", "127.0.0.1:8001", "Set listening address")
	user := flag.String("username", "", "Optional authentication username")
	password := flag.String("password", "", "Optional authentication password")
	flag.Parse()
	conf := &socks5.Config{
		Credentials: makeCredentials(*user, *password),
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}
	if err := server.ListenAndServe("tcp", *address); err != nil {
		panic(err)
	}
}
