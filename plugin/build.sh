#!/bin/sh

# generate govst.h
go build --buildmode=c-shared -o bridge.so bridge.go &&

# compile into shared library
go build -x --buildmode=c-shared -ldflags '-extldflags=-Wl,-soname,bridge.so' -o bridge.so
