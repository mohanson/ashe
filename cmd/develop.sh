set -ex

mkdir -p bin

go build -o bin github.com/mohanson/ashe/cmd/ashe
