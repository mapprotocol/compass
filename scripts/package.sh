#!/bin/bash

pushd `dirname $0` >/dev/null
cmdDir=$(pwd)
popd >/dev/null

cd "$cmdDir"/../target
rm -f *.zip
rm -f *.tar.gz
mv signmap_darwin_amd64.exe signmap
tar czvf signmap_darwin_amd64.tar.gz signmap

mv signmap_darwin_arm64.exe signmap
tar czvf signmap_darwin_arm64.tar.gz signmap

mv signmap_linux_amd64.exe signmap
tar czvf signmap_linux_amd64.tar.gz signmap

mv signmap_windows_amd64.exe signmap.exe
printf "signmap\\r\\npause" > signmap.bat
zip signmap_windows_amd64.zip signmap.exe
zip -u signmap_windows_amd64.zip signmap.bat
