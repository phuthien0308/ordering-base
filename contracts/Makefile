generate:
	protoc --proto_path=proto \
	       --go_out=gen --go_opt=paths=source_relative \
	       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
	       config/api.proto config/message.proto account/accountservice.proto account/address.proto
