#!/bin/bash

pushd `dirname $0` >/dev/null
cmddir=`pwd`
popd >/dev/null

target=(
  darwin,arm64
  darwin,amd64
  windows,amd64
  linux,amd64
)
targetdir=$cmddir/../target/
mkdir -p $targetdir
for i in "${target[@]}"
do
  pair=($(echo $i | tr ',' "\n"))
  GOOS=$pair
  GOARCH=${pair[1]}
  env GOOS=$GOOS GOARCH=$GOARCH go build -o $targetdir/signmap_"$GOOS"_"$GOARCH".exe $cmddir/../signmap/
done