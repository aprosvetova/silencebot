#!/bin/sh
platforms=("GOOS=linux GOARCH=amd64" "GOOS=linux GOARCH=386" "GOOS=windows GOARCH=amd64" "GOOS=windows GOARCH=386")
names=("linux-x64" "linux-i386" "win-x64.exe" "win-i386.exe")

echo "Resolving dependencies"

go get ./...

for i in "${!platforms[@]}"; do
	echo "Building ${names[$i]}"
	eval "CGO_ENABLED=0 ${platforms[$i]} go build -a -tags netgo -ldflags '-s -w -extldflags \"-static\"' -o bin/silencebot-${names[$i]}"
done

echo "Done"