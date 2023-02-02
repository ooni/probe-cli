# Directory MOBILE/android

This directory contains Android specific code. We will put
Android build artifacts inside it as well.

Scripts in this directory are generally invoked by:

* [../../Makefile](../../Makefile) for building mobile
releases (i.e., AAR files) out the
[../../pkg/oonimkall](../../pkg/oonimkall/) library;

* [../../MONOREPO/w](../../MONOREPO/w) workflows for
combining [../../pkg/oonimkall](../../pkg/oonimkall/)
builds with the Android app for immediate testing.

Code in [../../.github/workflows](../../.github/workflows/)
publishes artifacts in this directory when building tags.

These are the most important scripts:

* [adbinstall](adbinstall): installs a file named `app.apk`
in this directory into a phone connected via USB;

* [createpom](createpom): creates the `oonimkall.pom` file
from the `template.pom` template file;

* [ensure](ensure): ensures the required versions of SDK
tools and of the NDK are installed;

* [home](home): prints to stdout the location where the
Android SDK is installed (aka the `ANDROID_HOME`);

* [newkeystore](newkeystore): generates a `keystore.jks` file
suitable for signing testing `app.apk` files;

* [setup](setup): installs `sdkmanager` on Unix systems;

* [sign](sign): signs a file named `app-unsigned.apk` in this
directory using the `keystore.jks` keystore.
