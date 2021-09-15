# Tutorial: rewriting the torsf experiment

This tutorial teaches you how to write a minimal implementation of the
[torsf](https://github.com/ooni/spec/blob/master/nettests/ts-030-torsf.md)
experiment. We will do that in four steps.

In the [first step](chapter01/) we will write a `main.go`
function that runs the existing `torsf` implementation.

In the [second step](chapter02/) we will modify the existing
code to launch an empty experiment instead.

In the [third step](chapter03/) we will start to fill in
the empty experiment to more closely simulate a real implementation
of the `torsf` experiment.

In the [fourth step](chapter04/) we will replace the code
simulating a real `torsf` experiment with a minimal implementation
of such an experiment that uses other code in `ooni/probe-cli` to
attempt to bootstrap `tor` over Snowflake.

## Regenerating this tutorial

Most of the text of these tutorials comes from comments in real
Go code, to ensure that the code we show is always working against
the main development branch. For this reason, one should not edit
the README.md files manually when a Go file is also present in the
same directory. The following command regenerates all tutorials.

```
(cd ./internal/tutorial && go run ./generator)
```