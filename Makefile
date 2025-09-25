TAG ?= latest

generate_cert:
	@go run $(shell go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost --ca true
	@mkdir -p configs
	@mv cert.pem key.pem ./configs/
	@echo "certs generated at configs/cert.pem and configs/key.pem"

update_gomod:
	@echo "Updating go modules..."
	@go get -u ./...
	@go mod tidy

build_proto:
	@echo "Compiling protobuf: pkg/pb/governance.proto"
	@protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    pkg/pb/governance.proto

generate_docs:
	@npx @redocly/cli build-docs docs/public-api.yaml -o docs/api.html
	@echo "docs generated at docs/api.html"

mod_update_proto:
	@echo "Updating proto-related Go modules..."
	@go get google.golang.org/genproto@latest
	@go mod tidy

docker_build:
	@echo "Building Docker image: resolyte/service-governance:$(TAG)"
	@docker build -f Dockerfile.governance -t resolyte/service-governance:$(TAG) .

docker_push:
	@echo "Pushing Docker image: resolyte/service-governance:$(TAG)"
	@docker push resolyte/service-governance:$(TAG)

docker_release: docker_build docker_push
	@echo "released $(TAG)"

k8s_manifest:
	@echo "Generating combined manifest..."
	@rm -f k8s-manifest.yaml
	@for file in k8s/*.yaml; do \
		cat $$file >> k8s-manifest.yaml; \
		echo "" >> k8s-manifest.yaml; \
		echo "---" >> k8s-manifest.yaml; \
	done
	@echo "Combined manifest created at k8s-manifest.yaml"