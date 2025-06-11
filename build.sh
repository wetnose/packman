ver=$(cat version.txt)

rm -rf build

set -e
mkdir build

docker build \
  --no-cache --progress=plain --target=binaries --output=build \
  --build-arg VER=$ver \
  .

dir=$(pwd)

cd $dir/build/macos
zip ../packman-$ver-macos-x64.zip *

cd $dir/build/linux
zip ../packman-$ver-linux.zip *

cd $dir/build/windows
zip ../packman-$ver-windows.zip *