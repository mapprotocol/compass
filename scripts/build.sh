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
targets=(
  darwin,arm64
  darwin,amd64
  windows,amd64
  linux,amd64
)
targetDir=$cmdDir/../target/
mkdir -p "$targetDir"
((num=0))
for i in "${targets[@]}"
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
  env GOOS="$GOOS" GOARCH="$GOARCH" go build -o $targetDir/signmap_"$GOOS"_"$GOARCH".exe "$cmdDir"/../signmap/
done