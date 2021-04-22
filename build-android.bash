#!/bin/bash
set -ex
./build init
./build android --sign simone@openobservatory.org --bundle