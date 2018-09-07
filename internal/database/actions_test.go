package database

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
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
		t.Error(err)
	}
	result, err := CreateResult(sess, tmpdir, "websites", 0)
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

}
