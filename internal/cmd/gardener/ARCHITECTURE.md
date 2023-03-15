# Garderner architecture

The gardener is a batch tool. The `gardener sync` command creates a
local clone of the [test-lists](https://github.com/citizenlab/test-lists).

Once you have a local clone, you can execute commands to create reports
based on the [test-lists](https://github.com/citizenlab/test-lists) clone.

Commands are resumable. Each command creates a local sqlite database
containing all the work done so far. Therefore, you can always interrupt
a command and continue running it at a later time.

While running, the gardener uses the [aggregation API](
https://api.ooni.io/apidocs/#/default/get_api_v1_aggregation) to
collect statistics about specific URLs that we may want to
remove from the [test-lists](https://github.com/citizenlab/test-lists). To
avoid putting pressure on the [OONI API](https://api.ooni.io/), the
gardener SHOULD NOT run measurements in parallel.

When a command has finished running, it produces a summary CSV file
containing data useful to help test lists curator to take further
decisions regarding the changes to apply to the test lists. A command
that generates a report is usually named `<FOO>report`.

We also include commands that act upon generated reports by automatically
removing URL by applying _obvious_ rules. A researcher may still want
to remove additional URLs after that. A command that applies such rules
is usually named `<FOO>fix`.

