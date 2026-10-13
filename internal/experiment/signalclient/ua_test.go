package signalclient

import "testing"

func TestConfigUserAgent(t *testing.T) {
	cases := []struct {
		config Config
		hello  Hello
		want   string
	}{
		{Config{}, HelloAndroidWebSocket, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org)"},
		{Config{}, HelloLibsignal, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org) libsignal/0.105.0"},
		{Config{}, HelloLibsignalEnclave, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org) libsignal/0.105.0"},
		{Config{AppUserAgent: true}, HelloLibsignal, "Signal-Android/8.31.1 Android/34 libsignal/0.105.0"},
		{Config{AppUserAgent: true}, HelloDesktop, "Signal-Desktop/8.33.0 Linux"},
		{Config{UserAgent: "x", AppUserAgent: true}, HelloDesktop, "x"},
	}
	for _, c := range cases {
		if got := c.config.userAgent(c.hello); got != c.want {
			t.Errorf("%+v %s: got %q, want %q", c.config, c.hello.Name, got, c.want)
		}
	}
}
