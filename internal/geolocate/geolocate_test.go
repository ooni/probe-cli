package geolocate

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type taskProbeIPLookupper struct {
	ip  string
	err error
}

func (c taskProbeIPLookupper) LookupProbeIP(ctx context.Context) (string, error) {
	return c.ip, c.err
}

func TestLocationLookupCannotLookupProbeIP(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		probeIPLookupper: taskProbeIPLookupper{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != model.DefaultProbeASN {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != model.DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != model.DefaultProbeNetworkName {
		t.Fatal("invalid NetworkName value")
	}
	if out.IPAddr != model.DefaultProbeIP {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASNumber != model.DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIPAddr != model.DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverASNetworkName != model.DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

type taskASNLookupper struct {
	err  error
	asn  uint
	name string
}

func (c taskASNLookupper) LookupASN(ip string) (uint, string, error) {
	return c.asn, c.name, c.err
}

func TestLocationLookupCannotLookupProbeASN(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		probeIPLookupper:  taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper: taskASNLookupper{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != model.DefaultProbeASN {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != model.DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != model.DefaultProbeNetworkName {
		t.Fatal("invalid NetworkName value")
	}
	if out.IPAddr != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASNumber != model.DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIPAddr != model.DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverASNetworkName != model.DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

type taskCCLookupper struct {
	err error
	cc  string
}

func (c taskCCLookupper) LookupCC(ip string) (string, error) {
	return c.cc, c.err
}

func TestLocationLookupCannotLookupProbeCC(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		probeIPLookupper:  taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper: taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:  taskCCLookupper{cc: "US", err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != 1234 {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != model.DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != "1234.com" {
		t.Fatal("invalid NetworkName value")
	}
	if out.IPAddr != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASNumber != model.DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIPAddr != model.DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverASNetworkName != model.DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

type taskResolverIPLookupper struct {
	ip  string
	err error
}

func (c taskResolverIPLookupper) LookupResolverIP(ctx context.Context) (string, error) {
	return c.ip, c.err
}

func TestLocationLookupCannotLookupResolverIP(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		probeIPLookupper:    taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper:   taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:    taskCCLookupper{cc: "IT"},
		resolverIPLookupper: taskResolverIPLookupper{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if err != nil {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != 1234 {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != "IT" {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != "1234.com" {
		t.Fatal("invalid NetworkName value")
	}
	if out.IPAddr != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.didResolverLookup != true {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASNumber != model.DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIPAddr != model.DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverASNetworkName != model.DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestLocationLookupCannotLookupResolverNetworkName(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		probeIPLookupper:     taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper:    taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:     taskCCLookupper{cc: "IT"},
		resolverIPLookupper:  taskResolverIPLookupper{ip: "4.3.2.1"},
		resolverASNLookupper: taskASNLookupper{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if err != nil {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != 1234 {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != "IT" {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != "1234.com" {
		t.Fatal("invalid NetworkName value")
	}
	if out.IPAddr != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.didResolverLookup != true {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASNumber != model.DefaultResolverASN {
		t.Fatalf("invalid ResolverASN value: %+v", out.ResolverASNumber)
	}
	if out.ResolverIPAddr != "4.3.2.1" {
		t.Fatalf("invalid ResolverIP value: %+v", out.ResolverIPAddr)
	}
	if out.ResolverASNetworkName != model.DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestLocationLookupSuccessWithResolverLookup(t *testing.T) {
	op := Task{
		probeIPLookupper:     taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper:    taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:     taskCCLookupper{cc: "IT"},
		resolverIPLookupper:  taskResolverIPLookupper{ip: "4.3.2.1"},
		resolverASNLookupper: taskASNLookupper{asn: 4321, name: "4321.com"},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if err != nil {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != 1234 {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != "IT" {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != "1234.com" {
		t.Fatal("invalid NetworkName value")
	}
	if out.IPAddr != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.didResolverLookup != true {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASNumber != 4321 {
		t.Fatalf("invalid ResolverASN value: %+v", out.ResolverASNumber)
	}
	if out.ResolverIPAddr != "4.3.2.1" {
		t.Fatalf("invalid ResolverIP value: %+v", out.ResolverIPAddr)
	}
	if out.ResolverASNetworkName != "4321.com" {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestSmoke(t *testing.T) {
	config := Config{}
	task := NewTask(config)
	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non nil result")
	}
	// we already checked above that the returned
	// value is okay for all codepaths.
}

func TestASNStringWorks(t *testing.T) {
	r := Results{ASN: 1234}
	if r.ProbeASNString() != "AS1234" {
		t.Fatal("unexpected result")
	}
}
