generate:
	protoc --proto_path=. \
	       --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       account/accountservice.proto account/address.proto \
	       config/api.proto config/message.proto \
	       product/productservice.proto
