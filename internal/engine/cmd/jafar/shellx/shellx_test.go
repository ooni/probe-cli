package shellx

import "testing"

func TestRun(t *testing.T) {
	if err := Run("whoami"); err != nil {
		t.Fatal(err)
	}
	if err := Run("./nonexistent/command"); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestRunQuiet(t *testing.T) {
	if err := RunQuiet("true"); err != nil {
		t.Fatal(err)
	}
	if err := RunQuiet("./nonexistent/command"); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestRunCommandline(t *testing.T) {
	t.Run("when the command does not parse", func(t *testing.T) {
		if err := RunCommandline(`"foobar`); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when we have no arguments", func(t *testing.T) {
		if err := RunCommandline(""); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when we have a single argument", func(t *testing.T) {
		if err := RunCommandline("ls"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("when we have more than one argument", func(t *testing.T) {
		if err := RunCommandline("ls ."); err != nil {
			t.Fatal(err)
		}
	})
}
