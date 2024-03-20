SERVICE_NAME ?= oohelperd

ECS_CONTAINER_NAME ?= oonith-service-$(SERVICE_NAME)
IMAGE_NAME ?= ooni/oonith-$(SERVICE_NAME)
DATE := $(shell python3 -c "import datetime;print(datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%d'))")
GIT_FULL_SHA ?= $(shell git rev-parse HEAD)
SHORT_SHA := $(shell echo ${GIT_FULL_SHA} | cut -c1-8)
PKG_VERSION := "3.20.1"

BUILD_LABEL := $(DATE)-$(SHORT_SHA)
VERSION_LABEL = v$(PKG_VERSION)
ENV_LABEL ?= latest

print-labels:
	echo "ECS_CONTAINER_NAME=${ECS_CONTAINER_NAME}"
	echo "PKG_VERSION=${PKG_VERSION}"
	echo "BUILD_LABEL=${BUILD_LABEL}"
	echo "VERSION_LABEL=${VERSION_LABEL}"
	echo "ENV_LABEL=${ENV_LABEL}"

docker-build:
	# We need to use tar -czh to resolve the common dir symlink
	cd .. && docker build -f Dockerfile.oonith \
		--build-arg BRANCH_NAME=${VERSION_LABEL} \
		-t ${IMAGE_NAME}:${BUILD_LABEL} \
		-t ${IMAGE_NAME}:${VERSION_LABEL} \
		-t ${IMAGE_NAME}:${ENV_LABEL} \
		.

	echo "built image: ${IMAGE_NAME}:${BUILD_LABEL} (${IMAGE_NAME}:${VERSION_LABEL} ${IMAGE_NAME}:${ENV_LABEL})"

docker-push:
	# We need to use tar -czh to resolve the common dir symlink
	docker push ${IMAGE_NAME}:${BUILD_LABEL}
	docker push ${IMAGE_NAME}:${VERSION_LABEL}
	docker push ${IMAGE_NAME}:${ENV_LABEL}

docker-smoketest:
	echo "no smoketest implemented"

imagedefinitions.json:
	echo '[{"name":"${ECS_CONTAINER_NAME}","imageUri":"${IMAGE_NAME}:${BUILD_LABEL}"}]' > imagedefinitions.json

test:
	hatch run test

test-cov:
	hatch run test-cov

build:
	echo "no build implemented"

clean:
	rm -f imagedefinitions.json

run:
	cd .. && go run -tags netgo ./internal/cmd/oohelperd

.PHONY: init test build clean docker print-labels
