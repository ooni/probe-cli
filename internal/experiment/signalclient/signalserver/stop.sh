#!/bin/bash
# Stop the server started by run.sh (the Maven cache volume and the checkout stay for next time).
podman rm -f signal-server-local >/dev/null && echo stopped
