generate_cert:
	go run $(shell go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost --ca true
	mv cert.pem key.pem ./config/

update_gomod:
	go get -u ./...
	go mod tidy

build_proto:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    pkg/pb/governance.proto

mod_update_proto:
	go get google.golang.org/genproto@latest
	go mod tidy