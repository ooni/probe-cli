#!/usr/bin/env python3

""" This script takes in input the name of the tool to run followed by
    arguments and followed by the nettest name. The format recognized is
    the same of miniooni. Depending on the tool that we want to run, we
    reorder arguments so that they make sense for the tool.

    This is necessary because, albeit miniooni, MK, and OONI v2.x have
    more or less the same arguments, there are some differences. We could
    modify other tools to match miniooni, but this seems useless. """

import argparse
import os
import shlex
import sys

sys.path.insert(0, ".")
import common


def file_must_exist(pathname):
    """ Throws an exception if the given file does not actually exist. """
    if not os.path.isfile(pathname):
        raise RuntimeError("missing {}: please run miniooni first".format(pathname))
    return pathname


def main():
    apa = argparse.ArgumentParser()
    apa.add_argument("command", nargs=1, help="command to execute")

    # subset of arguments accepted by miniooni
    apa.add_argument(
        "-n", "--no-collector", action="count", help="don't submit measurement"
    )
    apa.add_argument("-o", "--reportfile", help="specify report file to use")
    apa.add_argument("-i", "--input", help="input for nettests taking an input")
    apa.add_argument("--home", help="override home directory")
    apa.add_argument("nettest", nargs=1, help="nettest to run")
    out = apa.parse_args()
    command, nettest = out.command[0], out.nettest[0]

    if "miniooni" not in command and "measurement_kit" not in command:
        raise RuntimeError("unrecognized tool")

    args = []
    args.append(command)
    if "miniooni" in command:
        args.extend(["--yes"])  # make sure we have informed consent
    if "measurement_kit" in command:
        args.extend(
            [
                "--ca-bundle-path",
                file_must_exist("{}/.miniooni/assets/ca-bundle.pem".format(out.home)),
            ]
        )
        args.extend(
            [
                "--geoip-country-path",
                file_must_exist("{}/.miniooni/assets/country.mmdb".format(out.home)),
            ]
        )
        args.extend(
            [
                "--geoip-asn-path",
                file_must_exist("{}/.miniooni/assets/asn.mmdb".format(out.home)),
            ]
        )
    if out.home and "miniooni" in command:
        args.extend(["--home", out.home])  # home applies to miniooni only
    if out.input:
        if "miniooni" in command:
            args.extend(["-i", out.input])  # input is -i for miniooni
    if out.no_collector:
        args.append("-n")
    if out.reportfile:
        args.extend(["-o", out.reportfile])
    args.append(nettest)
    if out.input and "measurement_kit" in command:
        if nettest == "web_connectivity":
            args.extend(["-u", out.input])  # MK's Web Connectivity uses -u for input

    sys.stderr.write("minioonilike.py: {}\n".format(shlex.join(args)))
    common.execute(args)


if __name__ == "__main__":
    main()
