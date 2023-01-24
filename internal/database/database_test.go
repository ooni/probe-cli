package database

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestConnect(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "dbtest")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tmpfile.Name())

	sess, err := Connect(tmpfile.Name())
	if err != nil {
		t.Error(err)
	}

	colls, err := sess.Collections()
	if err != nil {
		t.Error(err)
	}

	if len(colls) < 1 {
		t.Fatal("missing tables")
	}
}
