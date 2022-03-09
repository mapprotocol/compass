#!/bin/bash

pushd `dirname $0` >/dev/null
cmdDir=$(pwd)
popd >/dev/null

cd "$cmdDir"/../build
rm -f *.zip
rm -f *.tar.gz
mv compass_darwin_amd64 compass
tar czvf compass_darwin_amd64.tar.gz compass

mv compass_darwin_arm64 compass
tar czvf compass_darwin_arm64.tar.gz compass

mv compass_linux_amd64 compass
tar czvf compass_linux_amd64.tar.gz compass

mv compass_windows_amd64 compass.exe
printf "compass\\r\\npause" > compass.bat
zip compass_windows_amd64.zip compass.exe
zip -u compass_windows_amd64.zip compass.bat