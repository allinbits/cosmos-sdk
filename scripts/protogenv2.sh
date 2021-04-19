#!/bin/sh

set -e

proto_files=$(find ./apis -regex ".*\.\(proto\)")
ls -al ./third_party/proto/google
for file in $proto_files; do
  echo "building proto file $file"
  protoc -I=. -I=./third_party/proto --go_out=. --go-grpc_out=. "$file"
done

cp -r github.com/cosmos/cosmos-sdk/* ./
rm -rf github.com