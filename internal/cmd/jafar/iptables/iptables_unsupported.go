// +build !linux

package iptables

import "errors"

type otherwiseShell struct{}

func (*otherwiseShell) createChains() error {
	return errors.New("not implemented")
}
func (*otherwiseShell) dropIfDestinationEquals(ip string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) rstIfDestinationEqualsAndIsTCP(ip string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) dropIfContainsKeywordHex(keyword string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) dropIfContainsKeyword(keyword string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) rstIfContainsKeywordHexAndIsTCP(keyword string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) rstIfContainsKeywordAndIsTCP(keyword string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) hijackDNS(address string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) hijackHTTPS(address string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) hijackHTTP(address string) error {
	return errors.New("not implemented")
}
func (*otherwiseShell) waive() error {
	return errors.New("not implemented")
}

func newShell() *otherwiseShell {
	return &otherwiseShell{}
}
