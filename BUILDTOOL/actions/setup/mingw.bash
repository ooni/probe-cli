#help:
#help: Ensures mingw-w64 is installed and we're using the correct version.
#help:

# TODO: check for correct version

run $(command -v i686-w64-mingw32-gcc) --version | head -n1
run $(command -v x86_64-w64-mingw32-gcc) --version | head -n1
