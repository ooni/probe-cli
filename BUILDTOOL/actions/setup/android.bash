#help:
#help: Ensures that the Android SDK is installed at $ANDROID_SDK_DIR and
#help: that we've installed the correct version of several tools.
#help:
#help: Exports: ANDROID_HOME and ANDROID_NDK_HOME.
#help:

__action_setup_android() {
	local clitools_dir=$ANDROID_SDK_DIR/cmdline-tools/$ANDROID_CLITOOLS_VERSION
	local sdkmanager=$clitools_dir/bin/sdkmanager
	if [[ ! -x $sdkmanager ]]; then
		# Apparently, the Linux release works everywhere because Java is portable \m/
		local clitools_file="commandlinetools-linux-${ANDROID_CLITOOLS_VERSION}_latest.zip"
		local clitools_url="https://dl.google.com/android/repository/$clitools_file"
		run curl -fsSLO $clitools_url
		echo "$ANDROID_CLITOOLS_SHA256  $clitools_file" >SHA256SUMS
		run shasum -c SHA256SUMS
		run rm -rf $ANDROID_SDK_DIR
		run unzip $clitools_file
		run mkdir -p $ANDROID_SDK_DIR/cmdline-tools
		run mv cmdline-tools $clitools_dir
	fi
	run export ANDROID_HOME=$ANDROID_SDK_DIR
	echo "Yes" | run $sdkmanager --install "ndk;$ANDROID_NDK_VERSION"
	echo "Yes" | run $sdkmanager --install "build-tools;$ANDROID_BUILDTOOLS_VERSION"
	echo "Yes" | run $sdkmanager --install "platforms;$ANDROID_PLATFORM_VERSION"
	run export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/$ANDROID_NDK_VERSION
}

__action_setup_android
