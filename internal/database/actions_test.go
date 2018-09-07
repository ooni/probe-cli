package database

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/utils"
)

func TestMeasurementWorkflow(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "dbtest")
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("%s", tmpfile.Name())
	//defer os.Remove(tmpfile.Name())

	tmpdir, err := ioutil.TempDir("", "oonitest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	sess, err := Connect(tmpfile.Name())
	if err != nil {
		t.Error(err)
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
	err = sess.Collection("measurements").Find("id", m1.ID).One(&m2)
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
