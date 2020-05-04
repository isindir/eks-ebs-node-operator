SHELL := /bin/bash
GO 		:= GO15VENDOREXPERIMENT=1 GO111MODULE=on GOPROXY=https://proxy.golang.org go

IMAGE_NAME?="isindir/eks-ebs-node-operator"
SDK_IMAGE_NAME?="isindir/eks-ebs-node-operator-sdk"
VERSION?=$(shell awk 'BEGIN { FS=" = " } $$0~/Version = / \
				 { gsub(/"/, ""); print $$2; }' version/version.go)
BUILD:=`git rev-parse HEAD`
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

all: clean mod fmt check inspect build

.PHONY: mod
## mod: fetches dependencies
mod:
	@echo "Go Mod Vendor"
	$(GO) mod tidy
	$(GO) mod vendor
	@echo

.PHONY: echo
## echo: prints image name and version of the operator
echo:
	@echo "${IMAGE_NAME}:${VERSION}"
	@echo "${BUILD}"

.PHONY: clean
## clean: removes build artifacts from source code
clean:
	@echo "Cleaning"
	@rm -fr build/_output
	@rm -fr vendor
	@echo

.PHONY: inspect
## inspect: inspects remote docker 'image tag' - target fails if it does
inspect:
	@echo "Inspect remote image"
	@! DOCKER_CLI_EXPERIMENTAL="enabled" docker manifest inspect ${IMAGE_NAME}:${VERSION} >/dev/null \
		|| { echo "Image already exists"; exit 1; }

.PHONY: build
## build: builds operator docker image
build:
	@echo "Building"
	@operator-sdk build "${IMAGE_NAME}:${VERSION}"
	@docker tag "${IMAGE_NAME}:${VERSION}" "${IMAGE_NAME}:latest"
	@echo

.PHONY: build/sdk
## build/sdk: builds sdk docker image (not used)
build/sdk:
	@echo "Building sdk image"
	@docker build .circleci -t "${SDK_IMAGE_NAME}"
	@echo

.PHONY: push
## push: pushes operator docker image to repository
push:
	@echo "Pushing"
	@docker push "${IMAGE_NAME}:latest"
	@docker push "${IMAGE_NAME}:${VERSION}"
	@echo

.PHONY: push/sdk
## push/sdk: pushes sdk docker image to repository
push/sdk:
	@echo "Pushing"
	@docker push "${SDK_IMAGE_NAME}"
	@echo

.PHONY: fmt
## fmt: formats go code
fmt:
	@echo "Formatting"
	@gofmt -l -w $(SRC)
	@echo

.PHONY: check
## check: runs linting
check:
	@echo "Linting"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@echo

.PHONY: release
## release: builds operator docker image and pushes it to docker repository
release: build push

.PHONY: run/local
## run/local: runs operator in local mode
run/local:
	@OPERATOR_NAME=eks-ebs-node-operator WATCH_NAMESPACE="" operator-sdk run --local

.PHONY: cluster/create
## cluster/create: creates kind cluster and adds test label to a node
cluster/create:
	@kind create cluster --name operator
	@kubectl label nodes operator-control-plane beta.kubernetes.io/instance-type=m5a.2xlarge

.PHONY: cluster/delete
## cluster/delete: deletes kind cluster
cluster/delete:
	@kind delete cluster --name operator

.PHONY: run/sdk
## run/sdk: runs sdk docker image
run/sdk:
	@docker run -v ~/.gitconfig:/root/.gitconfig \
		-v ~/.gnupg:/root/.gnupg \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ${PWD}:/go/src/github.com/isindir/eks-ebs-node-operator \
		-ti "${SDK_IMAGE_NAME}" bash

.PHONY: help
## help: prints this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
