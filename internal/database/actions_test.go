package database

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ooni/probe-cli/utils"
)

func TestMeasurementWorkflow(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "dbtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpdir, err := ioutil.TempDir("", "oonitest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	sess, err := Connect(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	location := utils.LocationInfo{
		ASN:         0,
		CountryCode: "IT",
		NetworkName: "Unknown",
	}
	network, err := CreateNetwork(sess, &location)
	if err != nil {
		t.Fatal(err)
	}

	result, err := CreateResult(sess, tmpdir, "websites", network.ID)
	if err != nil {
		t.Fatal(err)
	}

	reportID := sql.NullString{String: "", Valid: false}
	testName := "antani"
	resultID := result.ID
	reportFilePath := tmpdir
	urlID := sql.NullInt64{Int64: 0, Valid: false}

	m1, err := CreateMeasurement(sess, reportID, testName, resultID, reportFilePath, urlID)
	if err != nil {
		t.Fatal(err)
	}

	var m2 Measurement
	err = sess.Collection("measurements").Find("measurement_id", m1.ID).One(&m2)
	if err != nil {
		t.Fatal(err)
	}
	if m2.ResultID != m1.ResultID {
		t.Error("result_id mismatch")
	}

	done, incomplete, err := ListResults(sess)
	if err != nil {
		t.Fatal(err)
	}

	if len(incomplete) != 1 {
		t.Error("there should be 1 incomplete measurement")
	}
	if len(done) != 0 {
		t.Error("there should be 0 done measurements")
	}

	msmts, err := ListMeasurements(sess, resultID)
	if err != nil {
		t.Fatal(err)
	}
	if msmts[0].Network.NetworkType != "wifi" {
		t.Error("network_type should be wifi")
	}
}

func TestDeleteResult(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "dbtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpdir, err := ioutil.TempDir("", "oonitest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	sess, err := Connect(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	location := utils.LocationInfo{
		ASN:         0,
		CountryCode: "IT",
		NetworkName: "Unknown",
	}
	network, err := CreateNetwork(sess, &location)
	if err != nil {
		t.Fatal(err)
	}

	result, err := CreateResult(sess, tmpdir, "websites", network.ID)
	if err != nil {
		t.Fatal(err)
	}

	reportID := sql.NullString{String: "", Valid: false}
	testName := "antani"
	resultID := result.ID
	reportFilePath := tmpdir
	urlID := sql.NullInt64{Int64: 0, Valid: false}

	m1, err := CreateMeasurement(sess, reportID, testName, resultID, reportFilePath, urlID)
	if err != nil {
		t.Fatal(err)
	}

	var m2 Measurement
	err = sess.Collection("measurements").Find("measurement_id", m1.ID).One(&m2)
	if err != nil {
		t.Fatal(err)
	}
	if m2.ResultID != m1.ResultID {
		t.Error("result_id mismatch")
	}

	err = DeleteResult(sess, resultID)
	if err != nil {
		t.Fatal(err)
	}
	totalResults, err := sess.Collection("results").Find().Count()
	if err != nil {
		t.Fatal(err)
	}
	totalMeasurements, err := sess.Collection("measurements").Find().Count()
	if err != nil {
		t.Fatal(err)
	}
	if totalResults != 0 {
		t.Fatal("results should be zero")
	}
	if totalMeasurements != 0 {
		t.Fatal("measurements should be zero")
	}
}

func TestNetworkCreate(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "dbtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	sess, err := Connect(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	l1 := utils.LocationInfo{
		ASN:         2,
		CountryCode: "IT",
		NetworkName: "Antaninet",
	}

	l2 := utils.LocationInfo{
		ASN:         3,
		CountryCode: "IT",
		NetworkName: "Fufnet",
	}

	_, err = CreateNetwork(sess, &l1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = CreateNetwork(sess, &l2)
	if err != nil {
		t.Fatal(err)
	}

}

func TestURLCreation(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "dbtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	sess, err := Connect(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	newID1, err := CreateOrUpdateURL(sess, "https://google.com", "GMB", "XX")
	if err != nil {
		t.Fatal(err)
	}

	newID2, err := CreateOrUpdateURL(sess, "https://google.com", "SRCH", "XX")
	if err != nil {
		t.Fatal(err)
	}

	newID3, err := CreateOrUpdateURL(sess, "https://facebook.com", "GRP", "XX")
	if err != nil {
		t.Fatal(err)
	}

	newID4, err := CreateOrUpdateURL(sess, "https://facebook.com", "GMP", "XX")
	if err != nil {
		t.Fatal(err)
	}

	newID5, err := CreateOrUpdateURL(sess, "https://google.com", "SRCH", "XX")
	if err != nil {
		t.Fatal(err)
	}

	if newID2 != newID1 {
		t.Error("inserting the same URL with different category code should produce the same result")
	}

	if newID3 == newID1 {
		t.Error("inserting different URL should produce different ids")
	}

	if newID4 != newID3 {
		t.Error("inserting the same URL with different category code should produce the same result")
	}

	if newID5 != newID1 {
		t.Error("the ID of google should still be the same")
	}
}

func TestPerformanceTestKeys(t *testing.T) {
	var tk PerformanceTestKeys

	ndtS := "{\"download\":100.0,\"upload\":20.0,\"ping\":2.2}"
	dashS := "{\"median_bitrate\":102.0}"
	if err := json.Unmarshal([]byte(ndtS), &tk); err != nil {
		t.Fatal("failed to parse ndtS")
	}
	if err := json.Unmarshal([]byte(dashS), &tk); err != nil {
		t.Fatal("failed to parse dashS")
	}
	if tk.Bitrate != 102.0 {
		t.Fatalf("error Bitrate %f", tk.Bitrate)
	}
	if tk.Download != 100.0 {
		t.Fatalf("error Download %f", tk.Download)
	}
}
