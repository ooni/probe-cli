package signalclient

import "testing"

func TestConfigUserAgent(t *testing.T) {
	cases := []struct {
		config Config
		client Client
		want   string
	}{
		{Config{}, ClientAndroid, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org)"},
		{Config{}, ClientLibsignal, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org) libsignal/0.105.0"},
		{Config{AppUserAgent: true}, ClientLibsignal, "Signal-Android/8.31.1 Android/34 libsignal/0.105.0"},
		{Config{AppUserAgent: true}, ClientDesktop, "Signal-Desktop/8.33.0 Linux"},
		{Config{UserAgent: "x", AppUserAgent: true}, ClientDesktop, "x"},
	}
	for _, c := range cases {
		if got := c.config.userAgent(c.client); got != c.want {
			t.Errorf("%+v %+v: got %q, want %q", c.config, c.client, got, c.want)
		}
	}
}
