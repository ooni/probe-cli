# OONI Probe CLI v3.0.0

| Author       | Simone Basso |
|--------------|--------------|
| Last-Updated | 2019-10-30   |
| Status       | open         |

## Introduction

This document describes the design of v3.0.0 of the OONI
Probe CLI (command line interface). The fundamental design
choice is that the CLI is a tool that other programs can
call. For this reason, the CLI contain subcommands allowing
another program to initiate any user-facing action meaningful
in the OONIverse. In the same vein, there is a flag for
forcing the CLI to emit output in JSONL format (i.e. a JSON
document on every line). The main use case for this functionality
is to control the command line from the OONI Desktop app.

## Background

The [legacy OONI Probe CLI](https://github.com/ooni/probe-legacy)
is a Python program that exposes a web user interface. This design
has served us well for years, however, it is significantly less
user friendly than the mobile apps. Therefore, it complicates life
for Windows and macOS users that are not developers and want to
use OONI. Hence, the choice of providing [a more user friendly app
for such users, OONI Probe Desktop](
https://github.com/ooni/probe-desktop). We are building this app for
Windows, macOS and Linux using Electron. We are aiming to provide
the same user experience of the mobile apps to desktop users.

The initial design was to link our C++ measurement library,
[Measurement Kit](https://github.com/measurement-kit/measurement-kit),
aka MK, directly into the Electron app. Then, after some iteration,
we have chosen a significantly more modular approach. The desktop app's
main concern is to provide a pleasant UI. The CLI's main concern is
to perform measurements and allow to see measurement results. This
design choice was originally documented in the [Writing a modern
cross-platform desktop app](
https://ooni.org/post/writing-a-modern-cross-platform-desktop-app/)
blog post published on the OONI website.

This document exists to document the functionality that the OONI
Probe CLI should expose to the desktop app, including the required
command line switches and the expected data format.

## Non goals

The main, initial design and implementation goal is to serve the
needs of the desktop app. This document currently does not address
the use cases of running OONI on Linux as a daemon. A future
version of this document is expected to address this limitation
and extend the interfaces defined here to cover such use case.

(If you have ideas on how that could be done, please contribute, to
this document explaining the use cases and proposing a specific
CLI addressing such use cases!)

## Use cases

We address two use cases: supporting the desktop app and running
the CLI as an interactive tool.

### Supporting the desktop app

Because we want the desktop app to provide the same user experience
of the mobile app, we need this functionality:

1. discover the user IP, country, and network

2. list the results stored on disk

3. delete one or more results

4. run any group of experiments (websites, instant messaging,
performance, middleboxes), or run them all

5. show each individual measurement

6. upload measurements that were not uploaded

To fully support this use case, we need to force the CLI to emit
messages in JSON, which is easily parseable from Electron.

### Interactive command line usage

We also want users to run the CLI interactively. Because we are
still working on the requirements of this use case, we will not be
entering into details as part of this document. A future version
will include more information.

## Software architecture

The following diagram summarizes the software architecture:

```
+--------------+   calls with flags   +----------+  .---------.
| OONI Desktop | -------------------> | OONI CLI | | local DB |
+--------------+                      +----------+  `---------'
          `--------<--<--<--<-------------'
                   JSON messages
```

The OONI CLI is a standalone, as static as possible, Go binary
that allows OONI Desktop to perform all the actions defined
above. The desktop app will communicate with the CLI using command
line flags. The OONI CLI will send _on the standard ouput_ a
stream of newline separated JSON messages (aka JSONL).

The OONI CLI keeps a local database keeping track of all the
network experiments that have been run. Measurements will
be saved on disk as JSON files.

The OONI CLI will statically link to Measurement Kit and/or
ooni/probe-engine, and/or any other library required to
perform OONI measurements.

The OONI CLI will be updated by the desktop app.

The OONI CLI will automatically download and store into
a configuration/state directory any resource that it
may require to perform its operations. This directory
is `$HOME/.ooni` on all systems.

The OONI CLI is a binary called `ooniprobe[.exe]`.

## Batch command line interface

This section describes the batch command line interface, i.e., the
one that should be used by the desktop app, or by any other tool, to
drive the OONI CLI.

The basic usage of the batch CLI is the following:

```
ooniprobe --batch --config <path> <command> [arguments]
```

The desktop app completely controls the current configuration
of the OONI CLI. Accordingly, it is expected to generate a
fresh configuration file for the CLI and pass it to the CLI
using `--config <path>`. This flag shall override any default
configuration file that the CLI would otherwise read.

The `--batch` flag makes the CLI emit JSON output. As said
above, this shall be emitted on the standard output.

### Terminology

An _experiment_ is a specific OONI experiment as codified in
the https://github.com/ooni/spec repository.

Users run _groups of experiments_. For example,`"im"` (short for Instant
Messaging), is the group of all OONI experiments that measure the
blocking of Instant Messagging apps.

A group of experiments produces a _result_.

Every result contains one or more _measurements_. The websites
group produce a measurement for every URL that was tested. All the
other groups produce a measurement for every experiment within
the group itself. For example, the performance group currently is
producing two measurements by default, generated by the NDT and
DASH experiments.

### Structure of most JSON messages

The general design is that most messages look like log messages
where specific information is represented as a JSON object under
the `fields` key. This way, a basic consumer of the output of
OONI CLI can just print the messages. More advanced consumers like
the desktop app will find actionable information in `fields`.

As you will see most JSON messages have this structure:

```JSON
{
  "fields": {
    "type": "engine"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:52:58.966368+02:00",
  "message": "(1) e14.whatsapp.net ipv4: 158.85.233.52"
}
```

where `fields` contains extra fields, `level` is the severity of
the message, `timestamp` is when it was emitted, and `message`
is a user-facing string. The `type` key within `fields` shall be
used by the desktop app to decide how to specifically process a
given JSON message.

### OONI desktop requirements

The desktop app MUST inspect all JSON messages and only process the
messages that it knows how to properly handle.

The order in which messages are received matters.

### Getting the CLI version

This command:

```
ooniprobe version
```

prints the version on the standard output, followed by newline.

### Automating the onboarding process

The desktop app is supposed to inform users about risks of
running OONI tests. The following command:

```
ooniprobe --batch --config <path> onboard --yes
```

forces the CLI to perform a batch onboarding process that
otherwise would have asked questions to users.

### Getting the current IP etc

The command

```
ooniprobe --batch --config <path> geoip
```

emits this output (where JSONs have been formatted for readability)

```JavaScript
// Header
{
  "fields": {
    "title": "GeoIP lookup",
    "type": "section_title"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:48:11.070934+02:00",
  "message": "GeoIP lookup"
}

// Result
{
  "fields": {
    "asn": "AS30722",
    "country_code": "IT",
    "ip": "127.0.0.1",
    "network_name": "Vodafone Italia S.p.A.",
    "type": "table"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:48:11.475014+02:00",
  "message": "Looked up your location"
}
```

### Listing all the results

This command:

```
ooniprobe --batch --config <path> list
```

lists on the standard output all the results.

In its most general form, the output of this command has
the following high level structure (where JSON messages have
been formatted for readability and we have added comments
to explain the meaning of fields):

```JavaScript
// section
{
  "fields": { "title": "Incomplete results", "type": "section_title" },
  "level": "info",
  "timestamp": "2019-10-03T16:25:25.393051+02:00",
  "message": "Incomplete results"
}

// zero or more failed results
{
  "fields": { /* [snip, same structure as below] */ },
  "level": "info",
  "timestamp": "2019-10-03T16:25:25.393307+02:00",
  "message":"result item"
}

// section
{
  "fields": { "title": "Results", "type": "section_title" },
  "level": "info",
  "timestamp": "2019-10-03T16:25:25.393358+02:00",
  "message": "Results"
}

// zero or more completed results
{
  "fields": {
    "asn": 30722,
    "data_usage_down": 2297.6103515625, // KiB
    "data_usage_up": 11.1669921875,     // KiB
    "id": 2,                            // result ID
    "index": 0,                         // index of entry in this output
    "is_done": true,
    "measurement_anomaly_count": 0,
    "measurement_count": 10,            // measurements in this result
    "name": "websites",                 // experiments group name
    "network_country_code": "IT",
    "network_name": "Vodafone Italia S.p.A.",
    "runtime": 0,
    "start_time": "2019-10-03T07:57:41.170538Z",
    "test_keys": "{}",                  // stuff to visualize immediately
    "total_count": 5,                   // entries in this output
    "type":"result_item"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:25:25.39541+02:00",
  "message": "result item"
}

/* [snip] */

// final message is summary
{
  "fields": {
    "total_data_usage_down": 223370.43359375,
    "total_data_usage_up": 24432.0224609375,
    "total_networks": 1,
    "total_tests": 5,
    "type": "result_summary",
  },
  "level": "info",
  "timestamp": "2019-10-03T16:25:25.397729+02:00",
  "message": "result summary"
}
```

The `id` field is the key to drill down the results.

### Listing measurements within a result

This command:

```
ooniprobe --batch --config <path> list <id>
```

provides information on the measurement within a specific result `<id>`.

In its most general form, the output of this command has
the following high level structure (where JSON messages have
been formatted for readability and we have added comments
to explain the meaning of fields):

```JavaScript
// entry describing a measurement
{
  "fields": {
    "asn": 30722,
    "failure_msg": "",
    "id": 125,                 // specific measurement ID
    "is_anomaly": false,
    "is_done": true,
    "is_failed": false,
    "is_first": true,
    "is_last": false,
    "is_upload_failed": false,
    "is_uploaded": true,
    "network_country_code": "IT",
    "network_name": "Vodafone Italia S.p.A.",
    "runtime": 0.317749,
    "start_time": "2019-10-03T07:59:10.325979Z",
    "test_group_name": "im",
    "test_keys": "{\"facebook_dns_blocking\":false,\"facebook_tcp_blocking\":false}",
    "test_name": "facebook_messenger",  // experiment name
    "type": "measurement_item",
    "upload_failure_msg": "",
    "url": "",
    "url_category_code": "",
    "url_country_code": ""
  },
  "level": "info",
  "timestamp": "2019-10-03T16:42:00.221666+02:00",
  "message": "measurement"
}

// [snip]

// final summary message
{
  "fields": {
    "anomaly_count": 0,
    "asn": 30722,
    "data_usage_down": 15.4765625,
    "data_usage_up": 4.556640625,
    "network_country_code": "IT",
    "network_name": "Vodafone Italia S.p.A.",
    "start_time": "2019-10-03T07:59:10.325979Z",
    "total_count": 3,
    "total_runtime": 3.141369,
    "type": "measurement_summary"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:42:00.22201+02:00",
  "message": "measurement summary"
}
```

### Getting a measurement JSON

This command:

```
ooniprobe --batch --config <path> show <id>
```

emits in output the measurement identified by `<id>`. The data
format is described at https://github.com/ooni/spec.

### Running experiments groups

This command:

```
ooniprobe --batch --config <path> run [all|im|middlebox|performance|websites]
```

runs the specified group of nettests, or all if no argument
is specified on the command line.

The general output format is the following (where again we
reformatted JSONs and added comments):

```JavaScript
// progress messages indicates the percentage progress. They are keyed
// by the experiments group followed by the experiment name.
{
  "fields": {
    "key": "im.FacebookMessenger",
    "percentage": 0.03333333333333333,
    "type": "progress"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:52:56.327636+02:00",
  "message": "starting the test"
}

// [snip]

// engine messages are log messages emitted by the measurement engine
{
  "fields": {
    "type": "engine"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:52:56.327828+02:00",
  "message": "starting facebook_messenger"
}

// [snip]

{
  "fields": {
    "key": "im.FacebookMessenger",
    "percentage": 0.3333333333333333,
    "type": "progress"
  },
  "level": "info",
  "timestamp": "2019-10-03T16:52:56.381365+02:00",
  "message": "test complete"
}

// [snip]
```

## State and configuration directory

This is `$HOME/.ooni` in all systems. It contains:

- `assets`: directory where assets like GeoIP files are downloaded

- `config.json`: default CLI config file, used unless you specify
another file using `--config <file>`

- `db`: directory containing the SQLite3 database

- `msmts`: directory containing the JSON measurements

Running concurrent instances of the OONI CLI is unsupported. They
may conflict attempting to download/update assets.

## Configuration file

The configuration file has the following structure:

```JSON
{
  "sharing": {
    "include_ip": false,
    "include_asn": true,
    "include_country": true,
    "upload_results": true
  },
  "test_settings": {
    "websites": {
      "enabled_categories": [],
      "limit": 0
    },
    "instant_messaging": {
      "enabled_tests": [
        "facebook-messenger",
        "whatsapp",
        "telegram"
      ]
    },
    "middlebox": {
      "enabled_tests": [
        "http-invalid-request-line",
        "http-header-field-manipulation"
      ]
    }
  }
}
```

where:

- `sharing` configures what information to include in measurements
and whether to automatically upload measurements

- `test_settings.websites` allows to configure the enabled
categories and how many URLs test at a time (`0` means no limit)

- `test_settings.instant_messaging` configures what experiments
to run when running the `im` group

- `test_settings.middlebox` configures what experiments
to run when running the `middlebox` group
