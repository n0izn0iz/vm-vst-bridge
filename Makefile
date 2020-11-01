gen:
	protoc --proto_path=api \
		--go_out=pkg/vstbridge --go_opt=paths=source_relative \
    --go-grpc_out=pkg/vstbridge --go-grpc_opt=paths=source_relative \
    vst_bridge.proto
