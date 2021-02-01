#!/bin/bash
set -e
pkgname=oonimkall
version=$(date -u +%Y.%m.%d-%H%M%S)
baseurl=https://api.bintray.com/content/ooni/android/$pkgname/$version/org/ooni/$pkgname/$version
aarfile=./MOBILE/android/$pkgname.aar
aarfile_version=./MOBILE/android/$pkgname-$version.aar
ln $aarfile $aarfile_version
sourcesfile=./MOBILE/android/$pkgname-sources.jar
sourcesfile_version=./MOBILE/android/$pkgname-$version-sources.jar
ln $sourcesfile $sourcesfile_version
pomfile=./MOBILE/android/$pkgname-$version.pom
pomtemplate=./MOBILE/template.pom
user=bassosimone
cat $pomtemplate|sed "s/@VERSION@/$version/g" > $pomfile
if [ -z $BINTRAY_API_KEY ]; then
    echo "FATAL: missing BINTRAY_API_KEY variable" 1>&2
    exit 1
fi
# We currently publish the mobile-staging branch. To cleanup we can fetch all the versions using
# the <curl -s $user:$BINTRAY_API_KEY https://api.bintray.com/packages/ooni/android/oonimkall>
# query, which returns a list of versions. From such list, we can delete the versions we
# don't need using <DELETE /packages/:subject/:repo/:package/versions/:version>.
for filename in $aarfile_version $sourcesfile_version $pomfile; do
  basefilename=$(basename $filename)
  curl -sT $filename -u $user:$BINTRAY_API_KEY $baseurl/$basefilename?publish=1 >/dev/null
done
echo "implementation 'org.ooni:oonimkall:$version'"
