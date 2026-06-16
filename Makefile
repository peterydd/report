# report - Makefile
# 高性能数据报表生成和发送系统

# 项目参数
BINARY_NAME=report
VERSION?=v1.1.0
DOCKER_IMAGE_NAME=peterydd/report
DOCKER_IMAGE_TAG=$(DOCKER_IMAGE_NAME):$(VERSION)
DOCKERFILE_PATH=./Dockerfile
BUILDPATH=./cmd/report

# Go 相关参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet
GOLINT=golangci-lint
GORUN=$(GOCMD) run

# 构建参数
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date +%Y-%m-%d-%H:%M:%S)"
CGO_ENABLED=0

# 颜色定义
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all
# 默认目标：构建项目
all: clean lint test build

# =============================================================================
# 开发相关
# =============================================================================

.PHONY: dev
# 开发模式：运行应用（不编译）
dev:
	@echo "$(GREEN)Running in development mode...$(NC)"
	$(GORUN) $(BUILDPATH)/main.go

.PHONY: build
# 构建二进制文件（带测试）
build: clean lint test
	@echo "$(GREEN)Building binary...$(NC)"
	cd $(BUILDPATH) && CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o ../../$(BINARY_NAME)
	@echo "$(GREEN)Build complete: $(BINARY_NAME)$(NC)"

.PHONY: build-skip-test
# 快速构建（跳过测试）
build-skip-test: clean
	@echo "$(YELLOW)Building binary (skip tests)...$(NC)"
	cd $(BUILDPATH) && CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o ../../$(BINARY_NAME)
	@echo "$(GREEN)Build complete: $(BINARY_NAME)$(NC)"

.PHONY: build-linux
# 构建 Linux 版本
build-linux: clean
	@echo "$(GREEN)Building for Linux AMD64...$(NC)"
	cd $(BUILDPATH) && CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o ../../$(BINARY_NAME)-linux-amd64

.PHONY: build-windows
# 构建 Windows 版本
build-windows: clean
	@echo "$(GREEN)Building for Windows AMD64...$(NC)"
	cd $(BUILDPATH) && CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o ../../$(BINARY_NAME)-windows-amd64.exe

.PHONY: build-all
# 构建所有平台版本
build-all: build-linux build-windows
	@echo "$(GREEN)All platform builds complete$(NC)"

.PHONY: run
# 构建并运行
run: build
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BINARY_NAME)

.PHONY: run-skip-test
# 快速构建并运行（跳过测试）
run-skip-test: build-skip-test
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BINARY_NAME)

# =============================================================================
# 测试相关
# =============================================================================

.PHONY: test
# 运行所有测试
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race ./...

.PHONY: test-short
# 运行简短测试
test-short:
	@echo "$(GREEN)Running short tests...$(NC)"
	$(GOTEST) -v -short ./...

.PHONY: test-coverage
# 运行测试并生成覆盖率报告
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

.PHONY: test-coverage-func
# 查看函数覆盖率
test-coverage-func: test-coverage
	$(GOCMD) tool cover -func=coverage.out

.PHONY: benchmark
# 运行基准测试
benchmark:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./...

# =============================================================================
# 代码质量
# =============================================================================

.PHONY: lint
# 运行代码检查
lint: fmt vet
	@echo "$(GREEN)Linting code...$(NC)"
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed, skipping...$(NC)"; \
	fi

.PHONY: fmt
# 格式化代码
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -w -s .

.PHONY: vet
# 静态代码检查
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) ./...

.PHONY: tidy
# 整理依赖
tidy:
	@echo "$(GREEN)Tidying modules...$(NC)"
	$(GOMOD) tidy
	$(GOMOD) verify

.PHONY: deps
# 下载依赖
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download

.PHONY: deps-update
# 更新依赖
deps-update:
	@echo "$(GREEN)Updating dependencies...$(NC)"
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

.PHONY: check
# 运行所有检查（lint + test）
check: lint test
	@echo "$(GREEN)All checks passed!$(NC)"

# =============================================================================
# 清理相关
# =============================================================================

.PHONY: clean
# 清理构建文件
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	$(GOCLEAN)
	@if exist report.exe ( del /F /Q report.exe ) else ( @rm -f report )
	@if exist report-*.exe ( del /F /Q report-*.exe ) else ( @rm -f report-* )
	@if exist report-windows-amd64.exe ( del /F /Q report-windows-amd64.exe )
	@if exist report-linux-amd64 ( del /F /Q report-linux-amd64 )
	@if exist *.xlsx ( del /F /Q *.xlsx ) else ( @rm -f *.xlsx )
	@if exist pkg\excel\*.xlsx ( del /F /Q pkg\excel\*.xlsx ) else ( @rm -f pkg/excel/*.xlsx )
	@if exist internal\app\*.xlsx ( del /F /Q internal\app\*.xlsx ) else ( @rm -f internal/app/*.xlsx )
	@if exist coverage.out ( del /F /Q coverage.out ) else ( @rm -f coverage.out )
	@if exist coverage.html ( del /F /Q coverage.html ) else ( @rm -f coverage.html )

.PHONY: clean-all
# 深度清理（包括Docker镜像）
clean-all: clean docker-clean
	@echo "$(GREEN)Deep clean complete$(NC)"

# =============================================================================
# E2E 相关（docker-compose + MySQL + MailHog）
# =============================================================================

COMPOSE_FILE=deploy/docker/docker-compose.e2e.yml
E2E_TAGS=e2e
E2E_TEST_PKGS=./test/e2e/...

.PHONY: e2e-up
# 启动 E2E 依赖栈（MySQL + MailHog）
e2e-up:
	@echo "$(GREEN)Starting E2E stack (mysql + mailhog)...$(NC)"
	docker compose -f $(COMPOSE_FILE) up -d
	@echo "$(YELLOW)Waiting for services to become healthy...$(NC)"
	@docker compose -f $(COMPOSE_FILE) ps

.PHONY: e2e-down
# 停止并移除 E2E 容器
e2e-down:
	@echo "$(YELLOW)Stopping E2E stack...$(NC)"
	docker compose -f $(COMPOSE_FILE) down

.PHONY: e2e-logs
# 查看 E2E 容器日志
e2e-logs:
	docker compose -f $(COMPOSE_FILE) logs -f

.PHONY: e2e-test
# 跑 E2E 测试（需先 make e2e-up）
e2e-test:
	@echo "$(GREEN)Running E2E tests...$(NC)"
	REPORT_INTEGRATION=1 REPORT_E2E=1 $(GOTEST) -count=1 -tags $(E2E_TAGS) $(E2E_TEST_PKGS)

.PHONY: e2e
# 完整 E2E：up → test → down
e2e: e2e-up
	@echo "$(GREEN)Waiting 5s for services to settle...$(NC)"
	@sleep 5
	@$(MAKE) e2e-test || ( $(MAKE) e2e-down && exit 1 )
	@$(MAKE) e2e-down
	@echo "$(GREEN)E2E complete.$(NC)"

.PHONY: e2e-clean
# 删除 E2E 卷与容器（包括 MySQL 数据）
e2e-clean:
	@echo "$(YELLOW)Removing E2E stack and volumes...$(NC)"
	docker compose -f $(COMPOSE_FILE) down -v

# =============================================================================
# Docker 相关
# =============================================================================

.PHONY: docker-build
# 构建 Docker 镜像
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -f $(DOCKERFILE_PATH) -t $(DOCKER_IMAGE_TAG) -t $(DOCKER_IMAGE_NAME):latest .

.PHONY: docker-build-no-cache
# 强制重新构建 Docker 镜像（无缓存）
docker-build-no-cache:
	@echo "$(GREEN)Building Docker image (no cache)...$(NC)"
	docker build --no-cache -f $(DOCKERFILE_PATH) -t $(DOCKER_IMAGE_TAG) .

.PHONY: docker-run
# 运行 Docker 容器
docker-run:
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run -d --name $(BINARY_NAME) -v $(PWD)/config.yaml:/config.yaml $(DOCKER_IMAGE_NAME):latest

.PHONY: docker-run-interactive
# 交互式运行 Docker 容器
docker-run-interactive:
	@echo "$(GREEN)Running Docker container (interactive)...$(NC)"
	docker run -it --rm -v $(PWD)/config.yaml:/config.yaml $(DOCKER_IMAGE_NAME):latest

.PHONY: docker-stop
# 停止 Docker 容器
docker-stop:
	@echo "$(YELLOW)Stopping Docker container...$(NC)"
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

.PHONY: docker-clean
# 清理 Docker 镜像
docker-clean:
	@echo "$(YELLOW)Cleaning Docker images...$(NC)"
	docker rmi -f $(DOCKER_IMAGE_TAG) || true
	docker rmi -f $(DOCKER_IMAGE_NAME):latest || true

.PHONY: docker-push
# 推送 Docker 镜像到仓库
docker-push: docker-build
	@echo "$(GREEN)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_NAME):latest

.PHONY: docker-scout
# 扫描 Docker 镜像安全漏洞
docker-scout: docker-build
	@echo "$(GREEN)Scanning Docker image...$(NC)"
	docker scout quickview $(DOCKER_IMAGE_TAG)

.PHONY: docker-logs
# 查看 Docker 日志
docker-logs:
	docker logs -f $(BINARY_NAME)

# =============================================================================
# 发布相关
# =============================================================================

.PHONY: version
# 显示版本信息
version:
	@echo "Version: $(VERSION)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Docker Image: $(DOCKER_IMAGE_TAG)"

.PHONY: changelog
# 生成变更日志（需要git-cliff）
changelog:
	@if command -v git-cliff > /dev/null; then \
		git-cliff -o CHANGELOG.md; \
	else \
		echo "$(YELLOW)git-cliff not installed, skipping...$(NC)"; \
	fi

.PHONY: tag
# 创建 Git 标签
tag:
	@echo "$(GREEN)Creating tag $(VERSION)...$(NC)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "$(YELLOW)Run 'git push origin $(VERSION)' to push the tag$(NC)"

.PHONY: release
# 完整发布流程
release: clean check build-all docker-build
	@echo "$(GREEN)Release $(VERSION) ready!$(NC)"
	@echo "Artifacts:"
	@echo "  - $(BINARY_NAME)-linux-amd64"
	@echo "  - $(BINARY_NAME)-windows-amd64.exe"
	@echo "  - Docker image: $(DOCKER_IMAGE_TAG)"

# =============================================================================
# 帮助
# =============================================================================

.PHONY: help
# 显示帮助信息
help:
	@echo ''
	@echo 'Usage: make [target] [VERSION=v1.x.x]'
	@echo ''
	@echo '$(GREEN)Development Targets:$(NC)'
	@echo '  $(YELLOW)make$(NC)                  运行所有检查并构建（默认）'
	@echo '  $(YELLOW)make dev$(NC)              开发模式运行'
	@echo '  $(YELLOW)make build$(NC)            构建二进制文件（带测试）'
	@echo '  $(YELLOW)make build-skip-test$(NC)  快速构建（跳过测试）'
	@echo '  $(YELLOW)make run$(NC)              构建并运行'
	@echo ''
	@echo '$(GREEN)Test Targets:$(NC)'
	@echo '  $(YELLOW)make test$(NC)             运行所有测试'
	@echo '  $(YELLOW)make test-coverage$(NC)    生成测试覆盖率报告'
	@echo '  $(YELLOW)make benchmark$(NC)        运行性能测试'
	@echo ''
	@echo '$(GREEN)Code Quality:$(NC)'
	@echo '  $(YELLOW)make lint$(NC)             运行代码检查'
	@echo '  $(YELLOW)make fmt$(NC)              格式化代码'
	@echo '  $(YELLOW)make check$(NC)            运行所有检查'
	@echo ''
	@echo '$(GREEN)Docker Targets:$(NC)'
	@echo '  $(YELLOW)make docker-build$(NC)     构建 Docker 镜像'
	@echo '  $(YELLOW)make docker-run$(NC)       运行 Docker 容器'
	@echo '  $(YELLOW)make docker-push$(NC)      推送 Docker 镜像'
	@echo '  $(YELLOW)make docker-scout$(NC)     扫描镜像安全漏洞'
	@echo ''
	@echo '$(GREEN)E2E Targets:$(NC)'
	@echo '  $(YELLOW)make e2e-up$(NC)           启动 MySQL + MailHog 容器'
	@echo '  $(YELLOW)make e2e-test$(NC)         跑 E2E Go 测试（需先 e2e-up）'
	@echo '  $(YELLOW)make e2e-down$(NC)         停止 E2E 容器'
	@echo '  $(YELLOW)make e2e$(NC)              up → test → down 完整流程'
	@echo '  $(YELLOW)make e2e-clean$(NC)        删除 E2E 容器与卷'
	@echo ''
	@echo '$(GREEN)Release Targets:$(NC)'
	@echo '  $(YELLOW)make release$(NC)          完整发布流程'
	@echo '  $(YELLOW)make version$(NC)          显示版本信息'
	@echo ''
	@echo '$(GREEN)Maintenance:$(NC)'
	@echo '  $(YELLOW)make clean$(NC)            清理构建文件'
	@echo '  $(YELLOW)make tidy$(NC)             整理依赖'
	@echo '  $(YELLOW)make deps-update$(NC)      更新依赖'
	@echo ''

.DEFAULT_GOAL := help
