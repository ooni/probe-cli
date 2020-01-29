# OONI Probe CLI

The next generation OONI Probe Command Line Interface.

## User setup

1. Go [into the releases](https://github.com/ooni/probe-cli/releases) and download the release for your architecture and platform

2. Extract the tarball with `tar xvzf ooniprobe_*.tar.gz`

3. Copy the `ooniprobe` binary into a location in your `$PATH`, for example `/usr/local/bin/ooniprobe`

4. Run `ooniprobe run` to perform all the tests

Optional:

Add a crontab entry (on linux) to run `ooniprobe` daily at a random time:

```
(crontab -l 2>/dev/null; echo "$(( ( RANDOM % 60 )  + 1 )) $(( ( RANDOM % 24 )  + 1 )) * * * ooniprobe run") | crontab -
```

On macOS you can configure OONI Probe to run automatically using launchd.

Below is a sample launchd script, that should be placed inside of `~/Library/LaunchAgents/org.ooni.probe.cli.plist`.

Be sure to replace `/PATH/TO/BINARY/ooniprobe` with the actual install location of the `ooniprobe` binary and `/PATH/TO/CONFIG/config-100sites.json` with the location of a file which limits the testing to 100 URLs.

You may also want to adjust the locations of the logs.

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>org.ooni.probe.daily-run</string>

    <key>KeepAlive</key>
    <false/>
    <key>RunAtLoad</key>
    <true/>

    <key>Program</key>
    <string>/PATH/TO/BINARY/ooniprobe</string>
    <key>ProgramArguments</key>
    <array>
        <string>--config="/PATH/TO/CONFIG/config-100sites.json"</string>
        <string>--batch</string>
        <string>run</string>
    </array>

    <key>StartInterval</key>
    <integer>3600</integer>

    <key>StandardErrorPath</key>
    <string>/tmp/ooniprobe-cli.err</string>

    <key>StandardOutPath</key>
    <string>/tmp/ooniprobe-cli.out</string>
</dict>
</plist>
```

## Development setup

Be sure you have golang >= 1.13. We use golang modules. Run

```
./build.sh help
```

to get information on the supported systems as well as to get
instructions on how to install dependencies.

## Updating dependencies

1. update every direct dependency in `go.mod` except `probe-engine`
using `go get -u -v $dependency`

2. pin to the latest version of the `probe-engine` with
`go get -v github.com/ooni/probe-engine@tag`

3. remove all indirect dependencies from `go.mod` and merge the
content of `probe-engine`'s `go.mod` into our `go.mod`

4. `go mod tidy`

The rationale of this procedure is that we want to pin exactly to
a specific version of psiphon and of its dependencies.

## Releasing

```
./build.sh release
```

and follow instructions.
