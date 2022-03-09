#!/bin/bash

pushd `dirname $0` >/dev/null
cmdDir=$(pwd)
popd >/dev/null
((l=0))
while getopts "l:" arg
do
  case $arg in
    l)
         ((l=OPTARG))
         ;;
    ?)
            echo "Unknown option"
            exit 1
            ;;
    esac
done
platforms=(
  darwin,arm64
  darwin,amd64
  windows,amd64
  linux,amd64
)
buildDir=$cmdDir/../build
mkdir -p "$buildDir"
((num=0))
for i in "${platforms[@]}"
do
  if [ "$l" -ne 0 ]  ; then
     ((num+=1))
     if  [ "$num" -ne "$l" ] ; then
        continue
     fi
  fi
  pair=($(echo "$i" | tr ',' "\n"))
  GOOS=$pair
  GOARCH=${pair[1]}
  echo "start building $GOOS-$GOARCH version..."
  env GOOS="$GOOS" GOARCH="$GOARCH" go build -o $buildDir/compass_"$GOOS"_"$GOARCH" "$cmdDir"/../cmd/compass/
  echo "$GOOS-$GOARCH version is built!"
done
