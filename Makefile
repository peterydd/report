# 项目参数
BINARY_NAME=report
DOCKER_IMAGE_NAME=peterydd/report
DOCKERFILE_PATH=./Dockerfile
BUILDPATH=./cmd/report

# Go 相关参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GORUN=$(GOCMD) run

# Docker 相关参数
DOCKER_BUILD_CMD=docker build -f $(DOCKERFILE_PATH) -t $(DOCKER_IMAGE_NAME) .

.PHONY: clean
# 清理构建文件
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) && rm -f *.xlsx && rm -f pkg/excel/*.xlsx && rm -f internal/report/*.xlsx

.PHONY: test
# 运行测试
test:
	$(GOTEST) -v ./...

.PHONY: build
# 构建二进制文件
build: clean test
	cd $(BUILDPATH) && $(GOBUILD) -o ../../$(BINARY_NAME)

.PHONY: build-skip-test
# 构建二进制文件
build-skip-test: clean
	cd $(BUILDPATH) && $(GOBUILD) -o ../../$(BINARY_NAME)

.PHONY: run
# 运行二进制文件
run: build
	chmod +x $(BINARY_NAME) && ./$(BINARY_NAME)

.PHONY: run-skip-test
# 运行二进制文件
run-skip-test: build-skip-test
	chmod +x $(BINARY_NAME) && ./$(BINARY_NAME)

.PHONY: docker-build
# 构建 Docker 镜像
docker-build: clean
	$(DOCKER_BUILD_CMD)

.PHONY: docker-run
# 运行 Docker 容器
docker-run:
	docker run -d $(DOCKER_IMAGE_NAME)

.PHONY: docker-clean
# 清理 Docker 镜像
docker-clean:
	docker rmi $(DOCKER_IMAGE_NAME)

.PHONY: docker-push
# 推送 Docker 镜像
docker-push:
	docker push $(DOCKER_IMAGE_NAME)

.PHONY: help
# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help