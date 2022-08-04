package main

//
// Session-related structs
//

func init() {
	// Note: this does not derive from BaseConfig because it's only used as an inner
	// member of other structs and cannot be passed directly to a task
	registerNewStruct(
		"Config for creating a new session",
		"SessionConfig",
		StructField{
			Docs: []string{
				"The verbosity level for the sessions's logger.",
				"",
				"It must be one of LogLevelDebug and LogLevelInfo or an empty string. In case",
				"it's an empty string, the code will assume LogLevelInfo.",
			},
			Name: "LogLevel",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The OPTIONAL probe-services URL.",
				"",
				"Leaving this field empty means we're going to use the default URL",
				"for communicating with the OONI backend. You may want to change",
				"this value for testing purposes or to use another backend.",
			},
			Name: "ProbeServicesURL",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The OPTIONAL proxy URL.",
				"",
				"Leaving this field empty means we're not using a proxy. You can",
				"use the following proxies:",
				"",
				"1. socks5://<host>:<port> to use a SOCKS5 proxy;",
				"",
				"2. tor:/// to launch the tor executable and use its SOCKS5 port;",
				"",
				"3. psiphon:/// to use the built-in Psiphon client as a proxy.",
				"",
				"On mobile devices, we will use a version of tor that we link as a library",
				"as opposed to using the tor executable. On desktop, you must have",
				"installed the tor executable somewhere in your PATH.",
			},
			Name: "ProxyURL",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The MANDATORY name of the tool using this library.",
				"",
				"You MUST specify this field or the session won't be started.",
			},
			Name: "SoftwareName",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The MANDATORY version of the tool using this library.",
				"",
				"You MUST specify this field or the session won't be started.",
			},
			Name: "SoftwareVersion",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The MANDATORY directory where to store the engine state.",
				"",
				"You MUST specify this field or the session won't be started.",
				"",
				"You MUST create this directory in advance.",
			},
			Name: "StateDir",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The MANDATORY directory where to store temporary files.",
				"",
				"You MUST specify this field or the session won't be started.",
				"",
				"You MUST create this directory in advance.",
				"",
				"The session will create a temporary directory _inside_ this directory",
				"and will remove the inner directory when it is finished running.",
			},
			Name: "TempDir",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"TorArgs contains OPTIONAL arguments to pass to tor.",
			},
			Name: "TorArgs",
			Type: TypeListString,
		},
		StructField{
			Docs: []string{
				"The OPTIONAL path to the tor binary.",
				"",
				"You can use this field to execute a version of tor that has",
				"not been installed inside your PATH.",
			},
			Name: "TorBinary",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The MANDATORY directory where to store persistent tunnel state.",
				"",
				"You MUST specify this field or the session won't be started.",
				"",
				"You MUST create this directory in advance.",
				"",
				"Both psiphon and tor will store information inside this directory when",
				"they're used as a circumention mechanism, i.e., using ProxyURL.",
			},
			Name: "TunnelDir",
			Type: TypeString,
		},
	)
}
