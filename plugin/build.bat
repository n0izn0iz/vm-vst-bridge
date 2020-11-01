:: # generate govst.h and compile into shared library
go build --buildmode=c-shared -o govst.dll plugin.go && go build -x --buildmode=c-shared -ldflags '-extldflags=-Wl,-soname,govst.dll' -o govst.dll