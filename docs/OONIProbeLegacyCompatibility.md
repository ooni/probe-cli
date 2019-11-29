# Command line flags

`-h, --help` Display help and exit
Supported: yes
probe-cli equivalent: `-h, --help`

`-n, --no-collector` Disable writing to collector
Supported: no
Priority: high

`-N, --no-njson` Disable writing to disk
Supported: no
Priority: low

`-g, --no-geoip` Disable geoip lookup on start.
Supported: no
Priority: low

`-s, --list` List the currently installed ooniprobe nettests
Supported: no
Priority: low

`-w, --web-ui` Start the web UI
Supported: no
Priority: wontfix, we have no web UI in probe-cli

`-z, --initialize` Initialize ooniprobe to begin running it
Supported: yes
probe-cli equivalent: `ooniprobe onboard`

`-o, --reportfile PATH_TO_FILE` Specify the report file name to write to.
Supported: no
Priority: medium

`-i, --testdeck PATH_TO_DECK` Specify as input a test deck: a yaml file containing the tests to run and their arguments.
Supported: no
Priority: wontfix, we have no deck support

`-c, --collector COLLECTOR_ADDRESS` Specify the address of the collector for test results. In most cases a user will prefer to specify a bouncer over this.
Supported: partially
probe-cli equivalent: edit ooniprobe.conf to specify the collector address in the options

`-b, --bouncer BOUNCER_ADDRESS` Specify the bouncer used to obtain the address of the collector and test helpers.
Supported: partially
probe-cli equivalent: edit ooniprobe.conf to specify the bouncer address in the options

`-l, --logfile PATH_TO_LOGFILE` Write to this logs to this filename.
Supported: no
Priority: medium

`-O, --pcapfile PATH_TO_PCAPFILE` Write a PCAP of the ooniprobe session to this filename.
Supported: no
Priority: wontfix, we don't have packet capture support in probe-cli

`-f, --configfile PATH_TO_CONFIG` Specify a path to the ooniprobe configuration file.
Supported: yes
probe-cli equivalent: `--config`

`-d, --datadir` Specify a path to the ooniprobe data directory.
Supported: yes
probe-cli equivalent: set the `OONI_HOME` environment variable

`-a, --annotations key:value[,key2:value2]` Annotate the report with a key:value[, key:value] format.
Supported: yes
Priority: high

`-P, --preferred-backend onion|https|cloudfront` Set the preferred backend to use when submitting results and/or communicating with test helpers. Can be either onion, https or cloudfront
Supported: no
Priority: wontfix, we don't support any other backend beyond https, yet we will
and yet we would prefer to have the logic of reporting be managed by the probe
itself and not expose this setting.

# Features

* Run a test deck
Supported: yes
probe-cli equivalent: we now call a test deck a test group and we have them
coded into the logic of the client

* Run an individual netttest
Supported: no
Priority: medium

* Upload a measurement like `oonireport upload`
Supported: no
Priority: medium

* Test an individual URL with web_connectivity
Supported: no
Priority: high

* Run tests automatically like `ooniprobe-agent`
Supported: no
Priority: high

* Write custom tests like OONI test templates
Supported: no
Priority: medium

* Packet captures
Supported: no
Priority: low

* Upload measurements using onion services
Supported: no
Priority: low

* Log level support to aid debugging
Supported: no
Priority: medium

* Measurement quota to limit the disk usage
Supported: no
Priority: high

* Failover strategies for uploading measurements (use https then onion then cloudfront)
Supported: no
Priority: high
