#!/bin/bash

pushd `dirname $0` >/dev/null
cmdDir=$(pwd)
popd >/dev/null

cd "$cmdDir"/../target

mv signmap_darwin_amd64.exe signmap
zip signmap_darwin_amd64.zip signmap

mv signmap_darwin_arm64.exe signmap
zip signmap_darwin_arm64.zip signmap

mv signmap_linux_amd64.exe signmap
zip signmap_linux_amd64.zip signmap

mv signmap_windows_amd64.exe signmap.exe
echo signmap > signmap.bat
zip signmap_windows_amd64.zip signmap.exe
zip -u signmap_windows_amd64.zip signmap.bat
