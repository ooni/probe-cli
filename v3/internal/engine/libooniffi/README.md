# Directory github.com/ooni/probe-engine/libooniffi

This directory contains code to generate shared/static libraries with
a Measurement Kit compatible ABI. To this end, we wrap the [oonimkall](
../oonimkall) API with a simple C API.

The generated libraries have a Measurement Kit compatible ABI. You can
also instruct your compiler so that [ooniffi.h](ooniffi.h) defines macros
that make code written for Measurement Kit compile and work. To this
end, please see comments insider [ooniffi.h](ooniffi.h).

To see how we compile this library for several systems, please take a
look at [libooniffi.yml](../.github/workflows/libooniffi.yml).

This is not used in any OONI product. We may break something
in ooniffi without noticing it. Please, be aware of that.
