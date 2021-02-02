#!/bin/bash
set -e
pkgname=oonimkall
version=$(date -u +%Y.%m.%d-%H%M%S)
baseurl=https://api.bintray.com/content/ooni/ios/$pkgname/$version
framework=./MOBILE/ios/$pkgname.framework
frameworkzip=./MOBILE/ios/$pkgname.framework.zip
podspecfile=./MOBILE/ios/$pkgname.podspec
podspectemplate=./MOBILE/template.podspec
user=bassosimone
(cd ./MOBILE/ios && rm -f $pkgname.framework.zip && zip -yr $pkgname.framework.zip $pkgname.framework)
cat $podspectemplate|sed "s/@VERSION@/$version/g" > $podspecfile
if [ -z $BINTRAY_API_KEY ]; then
    echo "FATAL: missing BINTRAY_API_KEY variable" 1>&2
    exit 1
fi
# We currently publish the mobile-staging branch. To cleanup we can fetch all the versions using
# the <curl -s $user:$BINTRAY_API_KEY https://api.bintray.com/packages/ooni/android/oonimkall>
# query, which returns a list of versions. From such list, we can delete the versions we
# don't need using <DELETE /packages/:subject/:repo/:package/versions/:version>.
curl -sT $frameworkzip -u $user:$BINTRAY_API_KEY $baseurl/$pkgname-$version.framework.zip?publish=1 >/dev/null
curl -sT $podspecfile -u $user:$BINTRAY_API_KEY $baseurl/$pkgname-$version.podspec?publish=1 >/dev/null
echo "pod 'oonimkall', :podspec => 'https://dl.bintray.com/ooni/ios/$pkgname-$version.podspec'"
