#!/bin/sh

BuildVersion=$(git tag | tail -n 1)
BuildTime=$(date +'%Y-%m-%d %H:%M:%S')
CommitHash=$(git rev-parse --short HEAD)

go build -ldflags "-X 'main.Tag=${BuildVersion}' -X 'main.BuildTime=${BuildTime}' -X 'main.CommitHash=$CommitHash'"
