#!/bin/bash

pushd `dirname $0` >/dev/null
cmdDir=$(pwd)
popd >/dev/null

cd "$cmdDir"/../target
rm -f *.zip
rm -f *.tar.gz
mv signmap_makalu_darwin_amd64.exe signmap_makalu
tar czvf signmap_makalu_darwin_amd64.tar.gz signmap_makalu

mv signmap_makalu_darwin_arm64.exe signmap_makalu
tar czvf signmap_makalu_darwin_arm64.tar.gz signmap_makalu

mv signmap_makalu_linux_amd64.exe signmap_makalu
strip signmap_makalu
tar czvf signmap_makalu_linux_amd64.tar.gz signmap_makalu

mv signmap_makalu_windows_amd64.exe signmap_makalu.exe
printf "signmap_makalu\\r\\npause" > signmap_makalu.bat
zip signmap_makalu_windows_amd64.zip signmap_makalu.exe
zip -u signmap_makalu_windows_amd64.zip signmap_makalu.bat
