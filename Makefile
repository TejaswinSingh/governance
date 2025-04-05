generate_cert:
	go run $(shell go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost --ca true
	mv cert.pem key.pem ./config/

update_gomod:
	go get -u ./...
	go mod tidy