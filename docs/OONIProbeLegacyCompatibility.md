# Command line flags

`-h, --help` Display help and exit
Supported: ✅
probe-cli equivalent: `-h, --help`

`-n, --no-collector` Disable writing to collector
Supported: ❌
Priority: high

`-N, --no-njson` Disable writing to disk
Supported: ❌
Priority: low

`-g, --no-geoip` Disable geoip lookup on start.
Supported: ❌
Priority: low

`-s, --list` List the currently installed ooniprobe nettests
Supported: ❌
Priority: low

`-w, --web-ui` Start the web UI
Supported: ❌
Priority: wontfix, we have no web UI in probe-cli

`-z, --initialize` Initialize ooniprobe to begin running it
Supported: ✅
probe-cli equivalent: `ooniprobe onboard`

`-o, --reportfile PATH_TO_FILE` Specify the report file name to write to.
Supported: ❌
Priority: medium

`-i, --testdeck PATH_TO_DECK` Specify as input a test deck: a yaml file containing the tests to run and their arguments.
Supported: ❌
Priority: wontfix, we have no deck support

`-c, --collector COLLECTOR_ADDRESS` Specify the address of the collector for test results. In most cases a user will prefer to specify a bouncer over this.
Supported: partially
probe-cli equivalent: edit ooniprobe.conf to specify the collector address in the options

`-b, --bouncer BOUNCER_ADDRESS` Specify the bouncer used to obtain the address of the collector and test helpers.
Supported: partially
probe-cli equivalent: edit ooniprobe.conf to specify the bouncer address in the options

`-l, --logfile PATH_TO_LOGFILE` Write to this logs to this filename.
Supported: ❌
Priority: medium

`-O, --pcapfile PATH_TO_PCAPFILE` Write a PCAP of the ooniprobe session to this filename.
Supported: ❌
Priority: wontfix, we don't have packet capture support in probe-cli

`-f, --configfile PATH_TO_CONFIG` Specify a path to the ooniprobe configuration file.
Supported: ✅
probe-cli equivalent: `--config`

`-d, --datadir` Specify a path to the ooniprobe data directory.
Supported: ✅
probe-cli equivalent: set the `OONI_HOME` environment variable

`-a, --annotations key:value[,key2:value2]` Annotate the report with a key:value[, key:value] format.
Supported: ✅
Priority: high

`-P, --preferred-backend onion|https|cloudfront` Set the preferred backend to use when submitting results and/or communicating with test helpers. Can be either onion, https or cloudfront
Supported: ❌
Priority: wontfix, we don't support any other backend beyond https, yet we will
and yet we would prefer to have the logic of reporting be managed by the probe
itself and not expose this setting.

# Features

* Run a test deck
Supported: ✅
probe-cli equivalent: we now call a test deck a test group and we have them
coded into the logic of the client

* Run an individual netttest
Supported: ❌
Priority: medium

* Upload a measurement like `oonireport upload`
Supported: ❌
Priority: medium

* Test an individual URL with web_connectivity
Supported: ❌
Priority: high

* Run tests automatically like `ooniprobe-agent`
Supported: ❌
Priority: high

* Write custom tests like OONI test templates
Supported: ❌
Priority: medium

* Packet captures
Supported: ❌
Priority: low

* Upload measurements using onion services
Supported: ❌
Priority: low

* Log level support to aid debugging
Supported: ❌
Priority: medium

* Measurement quota to limit the disk usage
Supported: ❌
Priority: high

* Failover strategies for uploading measurements (use https then onion then cloudfront)
Supported: ❌
Priority: high
