package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	db "upper.io/db.v3"
)

type locationInfo struct {
	asn         uint
	countryCode string
	ip          string
	networkName string
	resolverIP  string
}

func (lp *locationInfo) ProbeASN() uint {
	return lp.asn
}

func (lp *locationInfo) ProbeASNString() string {
	return fmt.Sprintf("AS%d", lp.asn)
}

func (lp *locationInfo) ProbeCC() string {
	return lp.countryCode
}

func (lp *locationInfo) ProbeIP() string {
	return lp.ip
}

func (lp *locationInfo) ProbeNetworkName() string {
	return lp.networkName
}

func (lp *locationInfo) ResolverIP() string {
	return lp.resolverIP
}

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

	location := locationInfo{
		asn:         0,
		countryCode: "IT",
		networkName: "Unknown",
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
	msmtFilePath := tmpdir
	urlID := sql.NullInt64{Int64: 0, Valid: false}

	m1, err := CreateMeasurement(sess, reportID, testName, msmtFilePath, 0, resultID, urlID)
	if err != nil {
		t.Fatal(err)
	}
	m1.IsUploaded = true
	m1.IsAnomaly = sql.NullBool{Valid: true, Bool: false}
	err = sess.Collection("measurements").Find("measurement_id", m1.ID).Update(m1)
	if err != nil {
		t.Fatal(err)
	}

	m2, err := CreateMeasurement(sess, reportID, testName, msmtFilePath, 0, resultID, urlID)
	if err != nil {
		t.Fatal(err)
	}
	m2.IsUploaded = false
	m2.IsAnomaly = sql.NullBool{Valid: true, Bool: true}
	err = sess.Collection("measurements").Find("measurement_id", m2.ID).Update(m2)
	if err != nil {
		t.Fatal(err)
	}

	if m2.ResultID != m1.ResultID {
		t.Error("result_id mismatch")
	}
	err = UpdateUploadedStatus(sess, result)
	if err != nil {
		t.Fatal(err)
	}
	result.Finished(sess)

	var r Result
	err = sess.Collection("measurements").Find("result_id", result.ID).One(&r)
	if err != nil {
		t.Fatal(err)
	}
	if r.IsUploaded == true {
		t.Error("result should be marked as not uploaded")
	}

	done, incomplete, err := ListResults(sess)
	if err != nil {
		t.Fatal(err)
	}

	if len(incomplete) != 0 {
		t.Error("there should be 0 incomplete result")
	}
	if len(done) != 1 {
		t.Error("there should be 1 done result")
	}

	if done[0].TotalCount != 2 {
		t.Error("there should be a total of 2 measurements in the result")
	}
	if done[0].AnomalyCount != 1 {
		t.Error("there should be a total of 1 anomalies in the result")
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

	location := locationInfo{
		asn:         0,
		countryCode: "IT",
		networkName: "Unknown",
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
	msmtFilePath := tmpdir
	urlID := sql.NullInt64{Int64: 0, Valid: false}

	m1, err := CreateMeasurement(sess, reportID, testName, msmtFilePath, 0, resultID, urlID)
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

	err = DeleteResult(sess, 20)
	if err != db.ErrNoMoreRows {
		t.Fatal(err)
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

	l1 := locationInfo{
		asn:         2,
		countryCode: "IT",
		networkName: "Antaninet",
	}

	l2 := locationInfo{
		asn:         3,
		countryCode: "IT",
		networkName: "Fufnet",
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

func TestGetMeasurementJSON(t *testing.T) {
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

	location := locationInfo{
		asn:         0,
		countryCode: "IT",
		networkName: "Unknown",
	}
	network, err := CreateNetwork(sess, &location)
	if err != nil {
		t.Fatal(err)
	}

	result, err := CreateResult(sess, tmpdir, "websites", network.ID)
	if err != nil {
		t.Fatal(err)
	}

	reportID := sql.NullString{String: "20210111T085144Z_ndt_RU_3216_n1_qMVnP0PTX7ObUSmD", Valid: true}
	testName := "antani"
	resultID := result.ID
	msmtFilePath := tmpdir
	urlID := sql.NullInt64{Int64: 0, Valid: false}

	msmt, err := CreateMeasurement(sess, reportID, testName, msmtFilePath, 0, resultID, urlID)
	if err != nil {
		t.Fatal(err)
	}
	msmt.IsUploaded = true
	err = sess.Collection("measurements").Find("measurement_id", msmt.ID).Update(msmt)
	if err != nil {
		t.Fatal(err)
	}

	tk, err := GetMeasurementJSON(sess, msmt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tk["probe_asn"] != "AS3216" {
		t.Error("inconsistent measurement downloaded")
	}
}
