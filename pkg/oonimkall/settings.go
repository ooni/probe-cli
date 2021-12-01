package oonimkall

// Settings contains settings for a task. This structure derives from
// the one described by MK v0.10.9 FFI API (https://git.io/Jv4Rv), yet
// since 2020-12-03 we're not backwards compatible anymore.
type settings struct {
	// Annotations contains the annotations to be added
	// to every measurements performed by the task.
	Annotations map[string]string `json:"annotations,omitempty"`

	// AssetsDir is the directory where to store assets. This
	// field is an extension of MK's specification. If
	// this field is empty, the task won't start.
	AssetsDir string `json:"assets_dir"`

	// DisabledEvents contains disabled events. See
	// https://git.io/Jv4Rv for the events names.
	//
	// This setting is currently ignored. We noticed the
	// code was ignoring it on 2021-12-01.
	DisabledEvents []string `json:"disabled_events,omitempty"`

	// Inputs contains the inputs. The task will fail if it
	// requires input and you provide no input.
	Inputs []string `json:"inputs,omitempty"`

	// LogLevel contains the logs level. See https://git.io/Jv4Rv
	// for the names of the available log levels.
	LogLevel string `json:"log_level,omitempty"`

	// Name contains the task name. By https://git.io/Jv4Rv the
	// names are in camel case, e.g. `Ndt`.
	Name string `json:"name"`

	// Options contains the task options.
	Options settingsOptions `json:"options"`

	// Proxy allows you to optionally force a specific proxy
	// rather than using no proxy (the default).
	//
	// Use `psiphon:///` to force using Psiphon with the
	// embedded configuration file. Not all builds have
	// an embedded configuration file, but OONI builds have
	// such a file, so they can use this functionality.
	//
	// Use `socks5://10.0.0.1:9050/` to connect to a SOCKS5
	// proxy running on 10.0.0.1:9050. This could be, for
	// example, a suitably configured `tor` instance.
	Proxy string

	// StateDir is the directory where to store persistent data. This
	// field is an extension of MK's specification. If
	// this field is empty, the task won't start.
	StateDir string `json:"state_dir"`

	// TempDir is the temporary directory. This field is an extension of MK's
	// specification. If this field is empty, we will pick the tempdir that
	// ioutil.TempDir uses by default, which may not work on mobile. According
	// to our experiments as of 2020-06-10, leaving the TempDir empty works
	// for iOS and does not work for Android.
	TempDir string `json:"temp_dir"`

	// TunnelDir is the directory where to store persistent state
	// related to circumvention tunnels. This directory is required
	// only if you want to use the tunnels. Added since 3.10.0.
	TunnelDir string `json:"tunnel_dir"`

	// Version indicates the version of this structure.
	Version int64 `json:"version"`
}

// settingsOptions contains the settings options
type settingsOptions struct {
	// MaxRuntime is the maximum runtime expressed in seconds. A negative
	// value for this field disables the maximum runtime. Using
	// a zero value will also mean disabled. This is not the
	// original behaviour of Measurement Kit, which used to run
	// for zero time in such case.
	MaxRuntime float64 `json:"max_runtime,omitempty"`

	// NoCollector indicates whether to use a collector
	NoCollector bool `json:"no_collector,omitempty"`

	// ProbeServicesBaseURL contains the probe services base URL.
	ProbeServicesBaseURL string `json:"probe_services_base_url,omitempty"`

	// SoftwareName is the software name. If this option is not
	// present, then the library startup will fail.
	SoftwareName string `json:"software_name,omitempty"`

	// SoftwareVersion is the software version. If this option is not
	// present, then the library startup will fail.
	SoftwareVersion string `json:"software_version,omitempty"`
}
