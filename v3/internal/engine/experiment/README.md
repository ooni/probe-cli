# Directory github.com/ooni/probe-engine/experiment

This directory contains the implementation of all the supported
experiments, one for each directory. The [OONI spec repository
contains a description of all the specified experiments](
https://github.com/ooni/spec/tree/master/nettests).

Note that in the OONI spec repository experiments are called
nettests. Originally, they were also called nettests here but
that created confusion with nettests in [ooni/probe-cli](
https://github.com/ooni/probe-cli). Therefore, we now use the
term experiment to indicate the implementation and the term
nettest to indicate the user facing view of such implementation.

Note that some experiments implemented here are not part of
the OONI specification. For example, the [urlgetter](urlgetter)
experiment is not in the OONI spec repository. The reason why
this happens is that `urlgetter` is an experiment "library" that
other experiments use to implement their functionality.

Likewise, the [example](example) experiment is a minimal
experiment that does nothing and you could use to bootstrap
the implementation of a new experiment. Of course, this
experiment is not part of the OONI specification.
