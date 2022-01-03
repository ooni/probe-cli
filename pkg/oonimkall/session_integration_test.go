package oonimkall_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/pkg/oonimkall"
)

func NewSessionForTestingWithAssetsDir(assetsDir string) (*oonimkall.Session, error) {
	return oonimkall.NewSession(&oonimkall.SessionConfig{
		AssetsDir:        assetsDir,
		ProbeServicesURL: "https://ams-pg-test.ooni.org/",
		SoftwareName:     "oonimkall-test",
		SoftwareVersion:  "0.1.0",
		StateDir:         "../testdata/oonimkall/state",
		TempDir:          "../testdata/",
	})
}

func NewSessionForTesting() (*oonimkall.Session, error) {
	return NewSessionForTestingWithAssetsDir("../testdata/oonimkall/assets")
}

func TestNewSessionWithInvalidStateDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := oonimkall.NewSession(&oonimkall.SessionConfig{
		StateDir: "",
	})
	if err == nil || !strings.HasSuffix(err.Error(), "no such file or directory") {
		t.Fatal("not the error we expected")
	}
	if sess != nil {
		t.Fatal("expected a nil Session here")
	}
}

func TestMaybeUpdateResourcesWithCancelledContext(t *testing.T) {
	// Note that MaybeUpdateResources is now a deprecated stub that
	// does nothing. We will remove it when we bump major.
	dir, err := ioutil.TempDir("", "xx")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sess, err := NewSessionForTestingWithAssetsDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	ctx.Cancel() // cause immediate failure
	err = sess.MaybeUpdateResources(ctx)
	// Explanation: we embed resources. We should change the API
	// and remove the context. Until we do that, let us just assert
	// that we have embedding and the context does not matter.
	if err != nil {
		t.Fatal(err)
	}
}

func ReduceErrorForGeolocate(err error) error {
	if err == nil {
		return errors.New("we expected an error here")
	}
	if errors.Is(err, context.Canceled) {
		return nil // when we have not downloaded the resources yet
	}
	if !errors.Is(err, geolocate.ErrAllIPLookuppersFailed) {
		return nil // otherwise
	}
	return fmt.Errorf("not the error we expected: %w", err)
}

func TestGeolocateWithCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	ctx.Cancel() // cause immediate failure
	location, err := sess.Geolocate(ctx)
	if err := ReduceErrorForGeolocate(err); err != nil {
		t.Fatal(err)
	}
	if location != nil {
		t.Fatal("expected nil location here")
	}
}

func TestGeolocateGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	location, err := sess.Geolocate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if location.ASN == "" {
		t.Fatal("location.ASN is empty")
	}
	if location.Country == "" {
		t.Fatal("location.Country is empty")
	}
	if location.IP == "" {
		t.Fatal("location.IP is empty")
	}
	if location.Org == "" {
		t.Fatal("location.Org is empty")
	}
}

func ReduceErrorForSubmitter(err error) error {
	if err == nil {
		return errors.New("we expected an error here")
	}
	if errors.Is(err, context.Canceled) {
		return nil // when we have not downloaded the resources yet
	}
	if err.Error() == "all available probe services failed" {
		return nil // otherwise
	}
	return fmt.Errorf("not the error we expected: %w", err)
}

func TestSubmitWithCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	ctx.Cancel() // cause immediate failure
	result, err := sess.Submit(ctx, "{}")
	if err := ReduceErrorForSubmitter(err); err != nil {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if result != nil {
		t.Fatal("expected nil result here")
	}
}

func TestSubmitWithInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	result, err := sess.Submit(ctx, "{")
	if err == nil || err.Error() != "unexpected end of JSON input" {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if result != nil {
		t.Fatal("expected nil result here")
	}
}

func DoSubmission(ctx *oonimkall.Context, sess *oonimkall.Session) error {
	inputm := model.Measurement{
		DataFormatVersion:    "0.2.0",
		MeasurementStartTime: "2019-10-28 12:51:07",
		MeasurementRuntime:   1.71,
		ProbeASN:             "AS30722",
		ProbeCC:              "IT",
		ProbeIP:              "127.0.0.1",
		ReportID:             "",
		ResolverIP:           "172.217.33.129",
		SoftwareName:         "miniooni",
		SoftwareVersion:      "0.1.0-dev",
		TestKeys:             map[string]bool{"success": true},
		TestName:             "example",
		TestVersion:          "0.1.0",
	}
	inputd, err := json.Marshal(inputm)
	if err != nil {
		return err
	}
	result, err := sess.Submit(ctx, string(inputd))
	if err != nil {
		return fmt.Errorf("session_test.go: submit failed: %w", err)
	}
	if result.UpdatedMeasurement == "" {
		return errors.New("expected non empty measurement")
	}
	if result.UpdatedReportID == "" {
		return errors.New("expected non empty report ID")
	}
	var outputm model.Measurement
	return json.Unmarshal([]byte(result.UpdatedMeasurement), &outputm)
}

func TestSubmitMeasurementGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	if err := DoSubmission(ctx, sess); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitCancelContextAfterFirstSubmission(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	if err := DoSubmission(ctx, sess); err != nil {
		t.Fatal(err)
	}
	ctx.Cancel() // fail second submission
	err = DoSubmission(ctx, sess)
	if err == nil || !strings.HasPrefix(err.Error(), "session_test.go: submit failed") {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestCheckInSuccess(t *testing.T) {
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	config := oonimkall.CheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		RunType:         "timed",
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: &oonimkall.CheckInConfigWebConnectivity{},
	}
	config.WebConnectivity.Add("NEWS")
	config.WebConnectivity.Add("CULTR")
	result, err := sess.CheckIn(ctx, &config)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if result == nil || result.WebConnectivity == nil {
		t.Fatal("got nil result or WebConnectivity")
	}
	if len(result.WebConnectivity.URLs) < 1 {
		t.Fatal("unexpected number of URLs")
	}
	if result.WebConnectivity.ReportID == "" {
		t.Fatal("got empty report ID")
	}
	siz := result.WebConnectivity.Size()
	if siz <= 0 {
		t.Fatal("unexpected number of URLs")
	}
	for idx := int64(0); idx < siz; idx++ {
		entry := result.WebConnectivity.At(idx)
		if entry.CategoryCode != "NEWS" && entry.CategoryCode != "CULTR" {
			t.Fatalf("unexpected category code: %+v", entry)
		}
	}
	if result.WebConnectivity.At(-1) != nil {
		t.Fatal("expected nil here")
	}
	if result.WebConnectivity.At(siz) != nil {
		t.Fatal("expected nil here")
	}
}

func TestCheckInLookupLocationFailure(t *testing.T) {
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	config := oonimkall.CheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		RunType:         "timed",
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: &oonimkall.CheckInConfigWebConnectivity{},
	}
	config.WebConnectivity.Add("NEWS")
	config.WebConnectivity.Add("CULTR")
	ctx.Cancel() // immediate failure
	result, err := sess.CheckIn(ctx, &config)
	if !errors.Is(err, geolocate.ErrAllIPLookuppersFailed) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if result != nil {
		t.Fatal("expected nil result here")
	}
}

func TestCheckInNewProbeServicesFailure(t *testing.T) {
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	sess.TestingCheckInBeforeNewProbeServicesClient = func(ctx *oonimkall.Context) {
		ctx.Cancel() // cancel execution
	}
	ctx := sess.NewContext()
	config := oonimkall.CheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		RunType:         "timed",
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: &oonimkall.CheckInConfigWebConnectivity{},
	}
	config.WebConnectivity.Add("NEWS")
	config.WebConnectivity.Add("CULTR")
	result, err := sess.CheckIn(ctx, &config)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if result != nil {
		t.Fatal("expected nil result here")
	}
}

func TestCheckInCheckInFailure(t *testing.T) {
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	sess.TestingCheckInBeforeCheckIn = func(ctx *oonimkall.Context) {
		ctx.Cancel() // cancel execution
	}
	ctx := sess.NewContext()
	config := oonimkall.CheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		RunType:         "timed",
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: &oonimkall.CheckInConfigWebConnectivity{},
	}
	config.WebConnectivity.Add("NEWS")
	config.WebConnectivity.Add("CULTR")
	result, err := sess.CheckIn(ctx, &config)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if result != nil {
		t.Fatal("expected nil result here")
	}
}

func TestCheckInNoParams(t *testing.T) {
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	config := oonimkall.CheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		RunType:         "timed",
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
	}
	result, err := sess.CheckIn(ctx, &config)
	if err == nil || err.Error() != "oonimkall: missing webconnectivity config" {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if result != nil {
		t.Fatal("unexpected not nil result here")
	}
}

func TestFetchURLListSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	config := oonimkall.URLListConfig{
		Limit: 10,
	}
	config.AddCategory("NEWS")
	config.AddCategory("CULTR")
	result, err := sess.FetchURLList(ctx, &config)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if result == nil || result.Results == nil {
		t.Fatal("got nil result")
	}
	for idx := int64(0); idx < result.Size(); idx++ {
		entry := result.At(idx)
		if entry.CategoryCode != "NEWS" && entry.CategoryCode != "CULTR" {
			t.Fatalf("unexpected category code: %+v", entry)
		}
	}
	if result.At(-1) != nil {
		t.Fatal("expected nil here")
	}
	if result.At(result.Size()) != nil {
		t.Fatal("expected nil here")
	}
}

func TestFetchURLListWithCC(t *testing.T) {
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	config := oonimkall.URLListConfig{
		CountryCode: "IT",
	}
	config.AddCategory("NEWS")
	config.AddCategory("CULTR")
	result, err := sess.FetchURLList(ctx, &config)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if result == nil || result.Results == nil {
		t.Fatal("got nil result")
	}
	found := false
	for _, entry := range result.Results {
		if entry.CountryCode == "IT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("not found url for country code: IT")
	}
}

func TestMain(m *testing.M) {
	// Here we're basically testing whether eventually the finalizers
	// will run and the number of active sessions and cancels will become
	// balanced. Especially for the number of active cancels, this is an
	// indication that we've correctly cleaned them up in the session.
	if exitcode := m.Run(); exitcode != 0 {
		os.Exit(exitcode)
	}
	for {
		runtime.GC()
		m, n := oonimkall.ActiveContexts.Load(), oonimkall.ActiveSessions.Load()
		fmt.Printf("./oonimkall: ActiveContexts: %d; ActiveSessions: %d\n", m, n)
		if m == 0 && n == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	os.Exit(0)
}
