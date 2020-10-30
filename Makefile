gen:
	protoc --go_out=vstbridge --go_opt=paths=source_relative \
    --go-grpc_out=vstbridge --go-grpc_opt=paths=source_relative \
    vst_bridge.proto
