// Code generated by go generate; DO NOT EDIT.
// 2021-03-10 12:20:36.438425974 +0100 CET m=+0.000181095

package ooapi

//go:generate go run ./internal/generator -file caching_test.go

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

func TestCachesimpleMeasurementMetaAPISuccess(t *testing.T) {
	ff := &fakeFill{}
	var expect *apimodel.MeasurementMetaResponse
	ff.fill(&expect)
	cache := &withCacheMeasurementMetaAPI{
		API: &FakeMeasurementMetaAPI{
			Response: expect,
		},
		KVStore: &memkvstore{},
	}
	var req *apimodel.MeasurementMetaRequest
	ff.fill(&req)
	ctx := context.Background()
	resp, err := cache.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if diff := cmp.Diff(expect, resp); diff != "" {
		t.Fatal(diff)
	}
}

func TestCachesimpleMeasurementMetaAPIWriteCacheError(t *testing.T) {
	errMocked := errors.New("mocked error")
	ff := &fakeFill{}
	var expect *apimodel.MeasurementMetaResponse
	ff.fill(&expect)
	cache := &withCacheMeasurementMetaAPI{
		API: &FakeMeasurementMetaAPI{
			Response: expect,
		},
		KVStore: &FakeKVStore{SetError: errMocked},
	}
	var req *apimodel.MeasurementMetaRequest
	ff.fill(&req)
	ctx := context.Background()
	resp, err := cache.Call(ctx, req)
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
}

func TestCachesimpleMeasurementMetaAPIFailureWithNoCache(t *testing.T) {
	errMocked := errors.New("mocked error")
	ff := &fakeFill{}
	cache := &withCacheMeasurementMetaAPI{
		API: &FakeMeasurementMetaAPI{
			Err: errMocked,
		},
		KVStore: &memkvstore{},
	}
	var req *apimodel.MeasurementMetaRequest
	ff.fill(&req)
	ctx := context.Background()
	resp, err := cache.Call(ctx, req)
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
}

func TestCachesimpleMeasurementMetaAPIFailureWithPreviousCache(t *testing.T) {
	ff := &fakeFill{}
	var expect *apimodel.MeasurementMetaResponse
	ff.fill(&expect)
	fakeapi := &FakeMeasurementMetaAPI{
		Response: expect,
	}
	cache := &withCacheMeasurementMetaAPI{
		API:     fakeapi,
		KVStore: &memkvstore{},
	}
	var req *apimodel.MeasurementMetaRequest
	ff.fill(&req)
	ctx := context.Background()
	// first pass with no error at all
	// use a separate scope to be sure we avoid mistakes
	{
		resp, err := cache.Call(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if diff := cmp.Diff(expect, resp); diff != "" {
			t.Fatal(diff)
		}
	}
	// second pass with failure
	errMocked := errors.New("mocked error")
	fakeapi.Err = errMocked
	fakeapi.Response = nil
	resp2, err := cache.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2 == nil {
		t.Fatal("expected non-nil response")
	}
	if diff := cmp.Diff(expect, resp2); diff != "" {
		t.Fatal(diff)
	}
}

func TestCachesimpleMeasurementMetaAPISetcacheWithEncodeError(t *testing.T) {
	ff := &fakeFill{}
	errMocked := errors.New("mocked error")
	var in []cacheEntryForMeasurementMetaAPI
	ff.fill(&in)
	cache := &withCacheMeasurementMetaAPI{
		GobCodec: &FakeCodec{EncodeErr: errMocked},
	}
	err := cache.setcache(in)
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestCachesimpleMeasurementMetaAPIReadCacheNotFound(t *testing.T) {
	ff := &fakeFill{}
	var incache []cacheEntryForMeasurementMetaAPI
	ff.fill(&incache)
	cache := &withCacheMeasurementMetaAPI{
		KVStore: &memkvstore{},
	}
	err := cache.setcache(incache)
	if err != nil {
		t.Fatal(err)
	}
	var req *apimodel.MeasurementMetaRequest
	ff.fill(&req)
	out, err := cache.readcache(req)
	if !errors.Is(err, errCacheNotFound) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestCachesimpleMeasurementMetaAPIWriteCacheDuplicate(t *testing.T) {
	ff := &fakeFill{}
	var req *apimodel.MeasurementMetaRequest
	ff.fill(&req)
	var resp1 *apimodel.MeasurementMetaResponse
	ff.fill(&resp1)
	var resp2 *apimodel.MeasurementMetaResponse
	ff.fill(&resp2)
	cache := &withCacheMeasurementMetaAPI{
		KVStore: &memkvstore{},
	}
	err := cache.writecache(req, resp1)
	if err != nil {
		t.Fatal(err)
	}
	err = cache.writecache(req, resp2)
	if err != nil {
		t.Fatal(err)
	}
	out, err := cache.readcache(req)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("expected non-nil here")
	}
	if diff := cmp.Diff(resp2, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestCachesimpleMeasurementMetaAPICacheSizeLimited(t *testing.T) {
	ff := &fakeFill{}
	cache := &withCacheMeasurementMetaAPI{
		KVStore: &memkvstore{},
	}
	var prev int
	for {
		var req *apimodel.MeasurementMetaRequest
		ff.fill(&req)
		var resp *apimodel.MeasurementMetaResponse
		ff.fill(&resp)
		err := cache.writecache(req, resp)
		if err != nil {
			t.Fatal(err)
		}
		out, err := cache.getcache()
		if err != nil {
			t.Fatal(err)
		}
		if len(out) > prev {
			prev = len(out)
			continue
		}
		break
	}
}
