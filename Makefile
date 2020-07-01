# Google Cloud
GCP_PROJECT?=tbd

# App specific
APP_NAME?=$(notdir $(shell pwd))
GITHUB_NAME?=$(shell cat go.mod | grep module | cut -d" " -f2)

# Docker
IMAGE=$(REGISTRY)/$(GITHUB_NAME)
REGISTRY?=eu.gcr.io/$(GCP_PROJECT)

# Builder settings
USE_DOCKER?=false
BUILD_TOOLS_VERSION=v8.4.3
BUILDER_IMAGE=gobuilder

PROTOC_VERSION_TAG=v1.3.1
GOPATH?=$(go env GOPATH)

# Build version
TAG_NAME?=$(shell git describe --tags 2> /dev/null || echo SNAPSHOT)
SHORT_SHA?=$(shell git rev-parse --short HEAD)
VERSION?=$(TAG_NAME)-$(SHORT_SHA)

# Go configs
ifeq ($(USE_DOCKER),false)
GOCMD=CGO_ENABLED=0 GOOS=linux go
LINTCMDROOT=CGO_ENABLED=0 GOOS=linux
else
GOCMD=docker run -it -e TAG_NAME="$(TAG_NAME)" -e SHORT_SHA="$(SHORT_SHA)"  -e CGO_ENABLED=0 -e GOOS=linux -e APP_NAME="$(APP_NAME)" --rm -w /go/in -v $(CURDIR):/go/in $(BUILDER_IMAGE) go
LINTCMDROOT=docker run -it --rm -w /go/in -v $(CURDIR):/go/in $(BUILDER_IMAGE)
endif


# Vendoring
# =======================

.PHONY: vendor
vendor: go.mod
	$(GOCMD) mod tidy
	$(GOCMD) mod vendor

.PHONY: vendor
vendor-commit: vendor
	git add ./vendor
	git commit -m "go mod & vendor"

# Testing
# =======================
.PHONY: lint
lint:
	$(LINTCMDROOT) golangci-lint "--config=./build/.golangci.yaml" run --fix

.PHONY: test
test: lint vendor test_all
	pass

.PHONY: test_all
test_all: test_unit test_integration
	pass

test_integration:
	$(GOCMD) test ./... -count=1 -tags=integration

test_unit:
	$(GOCMD) test ./... -count=1


# Building
# =======================
.PHONY: image_repository
image_repository:
	@echo $(IMAGE)

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: build
build:
	$(GOCMD) build -ldflags "-X main.Version=$(VERSION) -X main.Name=$(APP_NAME)" -o ./dist/go-app ./cmd/server

.PHONY: image
image:
	@echo "Building $(IMAGE):$(VERSION)"
	@docker build --build-arg=VERSION=$(VERSION) -f build/Dockerfile -t $(IMAGE):$(VERSION) .
	@docker tag $(IMAGE):$(VERSION)  $(IMAGE):latest


# Running
# =======================
run-docker: image
	@docker run $(IMAGE):$(VERSION)

run:
	$(GOCMD) run -ldflags "-X main.Version=$(VERSION) -X main.Name=$(APP_NAME)" ./cmd/server

# Release
# =======================
.PHONY: push
push: image
	@echo "Pushing $(IMAGE):$(VERSION)"
	@docker push $(IMAGE):$(VERSION)

# Deploying
# =======================
.PHONY: deploy-gcp-cloud-run
deploy-gcp-cloud-run: push
	gcloud run deploy --image $(IMAGE):$(VERSION) \
	    --project=$(GCP_PROJECT) \
		--platform managed \
		--max-instances=10 \
		--memory=128Mi \
		--update-env-vars "$(cat deploy.env | sed 's/export //g' | tr '\n' ',')"
		--service-account $(APP_NAME)@$(GCP_PROJECT).iam.gserviceaccount.com
# Cleaing
# =======================
clean:
	@$(GOCMD) clean
	@$(GOCMD) clean -modcache
	@rm -rf ./dist/* 2> /dev/null

distclean: clean
	@rm -rf vendor


## Service account
service-account:
	gcloud iam service-accounts create $(APP_NAME) \
    --description="service account for the $(APP_NAME) service" \
    --display-name="sa-$(APP_NAME)" \
	--project $(GCP_PROJECT)

bigquery-dataset:
	bq --location=EU mk \
	--dataset \
	--description description \
	$(GCP_PROJECT):tealium_export


#
# Development env setup
# =======================
dev-builderimg:
	@docker build -t $(BUILDER_IMAGE) -f build/Dockerfile --target BuildEnv .

dev-deps:
	#GO111MODULE=on go get -u github.com/golangci/golangci-lint/cmd/golangci-lint@v1.26.0
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.27.0

	curl -s -f -L -o await "https://github.com/betalo-sweden/await/releases/download/v0.4.0/await-linux-amd64"
	chmod +x await
	mv await /root/await

	go get -u -d golang.org/x/tools/cmd/goimports
	go install golang.org/x/tools/cmd/goimports
	go get -u -d github.com/golang/mock/gomock
	go install github.com/golang/mock/gomock
	go get -u -d github.com/golang/mock/mockgen
	go install github.com/golang/mock/mockgen

	# Workaround for https://github.com/golang/protobuf/issues/763#issuecomment-442767135.
	# Then `checkout master` because `go get` below does `pull --ff-only` in same local repo.
	go get -u -d github.com/golang/protobuf/protoc-gen-go
	git -C $(GOPATH)/src/github.com/golang/protobuf checkout $(PROTOC_VERSION_TAG)
	go install github.com/golang/protobuf/protoc-gen-go
	git -C $(GOPATH)/src/github.com/golang/protobuf checkout master

	go get -u -d github.com/golang/protobuf/ptypes
	git -C $(GOPATH)/src/github.com/golang/protobuf checkout $(PROTOC_VERSION_TAG)
	go install github.com/golang/protobuf/ptypes
	git -C $(GOPATH)/src/github.com/golang/protobuf checkout master

	go get -u google.golang.org/grpc github.com/kevinburke/go-bindata/...
	curl -sSL https://sdk.cloud.google.com | bash
