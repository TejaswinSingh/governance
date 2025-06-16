.PHONY: generate_cert update_gomod build_proto mod_update_proto docker_build docker_push docker_release

TAG ?= latest

generate_cert:
	go run $(shell go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost --ca true
	mv cert.pem key.pem ./configs/

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

docker_build:
	docker build -f Dockerfile.governance -t resolyte/service-governance:$(TAG) .

docker_push:
	docker push resolyte/service-governance:$(TAG)

docker_release: docker_build docker_push

k8s_manifest:
	@echo "Generating combined manifest..."
	@rm -f k8s-manifest.yaml
	@for file in k8s/*.yaml; do \
		cat $$file >> k8s-manifest.yaml; \
		echo "" >> k8s-manifest.yaml; \
		echo "---" >> k8s-manifest.yaml; \
	done
	@echo "Combined manifest created at k8s-manifest.yaml"