#!/bin/bash

CC=$(which musl-gcc) go build --ldflags '-w -linkmode external -extldflags "-static"' -o github-bot-alpine
