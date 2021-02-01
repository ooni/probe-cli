package geolocate

import (
	"context"
	"errors"
	"testing"
)

type taskResourcesManager struct {
	asnDatabasePath     string
	countryDatabasePath string
	err                 error
}

func (c taskResourcesManager) ASNDatabasePath() string {
	return c.asnDatabasePath
}

func (c taskResourcesManager) CountryDatabasePath() string {
	return c.countryDatabasePath
}

func (c taskResourcesManager) MaybeUpdateResources(ctx context.Context) error {
	return c.err
}

func TestLocationLookupCannotUpdateResources(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		resourcesManager: taskResourcesManager{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != DefaultProbeASN {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != DefaultProbeNetworkName {
		t.Fatal("invalid NetworkName value")
	}
	if out.ProbeIP != DefaultProbeIP {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIP != DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

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
		resourcesManager: taskResourcesManager{},
		probeIPLookupper: taskProbeIPLookupper{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != DefaultProbeASN {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != DefaultProbeNetworkName {
		t.Fatal("invalid NetworkName value")
	}
	if out.ProbeIP != DefaultProbeIP {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIP != DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

type taskASNLookupper struct {
	err  error
	asn  uint
	name string
}

func (c taskASNLookupper) LookupASN(path string, ip string) (uint, string, error) {
	return c.asn, c.name, c.err
}

func TestLocationLookupCannotLookupProbeASN(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		resourcesManager:  taskResourcesManager{},
		probeIPLookupper:  taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper: taskASNLookupper{err: expected},
	}
	ctx := context.Background()
	out, err := op.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out.ASN != DefaultProbeASN {
		t.Fatal("invalid ASN value")
	}
	if out.CountryCode != DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != DefaultProbeNetworkName {
		t.Fatal("invalid NetworkName value")
	}
	if out.ProbeIP != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIP != DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

type taskCCLookupper struct {
	err error
	cc  string
}

func (c taskCCLookupper) LookupCC(path string, ip string) (string, error) {
	return c.cc, c.err
}

func TestLocationLookupCannotLookupProbeCC(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		resourcesManager:  taskResourcesManager{},
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
	if out.CountryCode != DefaultProbeCC {
		t.Fatal("invalid CountryCode value")
	}
	if out.NetworkName != "1234.com" {
		t.Fatal("invalid NetworkName value")
	}
	if out.ProbeIP != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIP != DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
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
		resourcesManager:     taskResourcesManager{},
		probeIPLookupper:     taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper:    taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:     taskCCLookupper{cc: "IT"},
		resolverIPLookupper:  taskResolverIPLookupper{err: expected},
		enableResolverLookup: true,
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
	if out.ProbeIP != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.DidResolverLookup != true {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatal("invalid ResolverASN value")
	}
	if out.ResolverIP != DefaultResolverIP {
		t.Fatal("invalid ResolverIP value")
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestLocationLookupCannotLookupResolverNetworkName(t *testing.T) {
	expected := errors.New("mocked error")
	op := Task{
		resourcesManager:     taskResourcesManager{},
		probeIPLookupper:     taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper:    taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:     taskCCLookupper{cc: "IT"},
		resolverIPLookupper:  taskResolverIPLookupper{ip: "4.3.2.1"},
		resolverASNLookupper: taskASNLookupper{err: expected},
		enableResolverLookup: true,
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
	if out.ProbeIP != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.DidResolverLookup != true {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatalf("invalid ResolverASN value: %+v", out.ResolverASN)
	}
	if out.ResolverIP != "4.3.2.1" {
		t.Fatalf("invalid ResolverIP value: %+v", out.ResolverIP)
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestLocationLookupSuccessWithResolverLookup(t *testing.T) {
	op := Task{
		resourcesManager:     taskResourcesManager{},
		probeIPLookupper:     taskProbeIPLookupper{ip: "1.2.3.4"},
		probeASNLookupper:    taskASNLookupper{asn: 1234, name: "1234.com"},
		countryLookupper:     taskCCLookupper{cc: "IT"},
		resolverIPLookupper:  taskResolverIPLookupper{ip: "4.3.2.1"},
		resolverASNLookupper: taskASNLookupper{asn: 4321, name: "4321.com"},
		enableResolverLookup: true,
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
	if out.ProbeIP != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.DidResolverLookup != true {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASN != 4321 {
		t.Fatalf("invalid ResolverASN value: %+v", out.ResolverASN)
	}
	if out.ResolverIP != "4.3.2.1" {
		t.Fatalf("invalid ResolverIP value: %+v", out.ResolverIP)
	}
	if out.ResolverNetworkName != "4321.com" {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestLocationLookupSuccessWithoutResolverLookup(t *testing.T) {
	op := Task{
		resourcesManager:     taskResourcesManager{},
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
	if out.ProbeIP != "1.2.3.4" {
		t.Fatal("invalid ProbeIP value")
	}
	if out.DidResolverLookup != false {
		t.Fatal("invalid DidResolverLookup value")
	}
	if out.ResolverASN != DefaultResolverASN {
		t.Fatalf("invalid ResolverASN value: %+v", out.ResolverASN)
	}
	if out.ResolverIP != DefaultResolverIP {
		t.Fatalf("invalid ResolverIP value: %+v", out.ResolverIP)
	}
	if out.ResolverNetworkName != DefaultResolverNetworkName {
		t.Fatal("invalid ResolverNetworkName value")
	}
}

func TestSmoke(t *testing.T) {
	maybeFetchResources(t)
	config := Config{
		EnableResolverLookup: true,
		ResourcesManager: taskResourcesManager{
			asnDatabasePath:     asnDBPath,
			countryDatabasePath: countryDBPath,
		},
	}
	task := Must(NewTask(config))
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

func TestNewTaskWithNoResourcesManager(t *testing.T) {
	task, err := NewTask(Config{})
	if !errors.Is(err, ErrMissingResourcesManager) {
		t.Fatal("not the error we expected")
	}
	if task != nil {
		t.Fatal("expected nil task here")
	}
}
