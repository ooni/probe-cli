#!/bin/bash
set -euxo pipefail
gcc -g -Wall -Wextra -fsanitize=thread -I internal/libtor/linux/amd64/include -L internal/libtor/linux/amd64/lib y/main.c -ltor -levent -lssl -lcrypto -lz -lm
