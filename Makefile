.PHONY: build test vet image

build:
	go build -o sealix ./cmd/sealix

test:
	go test ./..

vet:
	go vet ./.. 

image:
	docker build --build-arg VERSION=dev -t sealix:dev ,
