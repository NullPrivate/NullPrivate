# Repository Guidelines

## 项目结构与模块组织

- `main.go` / `main_next.go`：服务入口与启动逻辑。
- `internal/`：Go 后端核心实现（业务、DNS、存储、API 等）。
- `client/`：Web Dashboard（React + TypeScript + Webpack）。
- `scripts/`、`scripts/make/`：构建/CI 脚本与工具（建议先运行 `make init` 配置 Git hooks）。
- `dist/`：构建产物输出目录（可由 `DIST_DIR` 覆盖）。
- `docker/`、`services/`、`snap/`：部署与打包相关内容。
- `openapi/`、`doc/`、`readmes/`：API 与文档资源。

## 开发环境

- Go 版本以 `go.mod` 为准（当前为 `go 1.26.0`）；CI 工作流中的 `GO_VERSION` 需不低于该版本；如需查看构建环境变量可用 `make go-env`。
- 前端依赖位于 `client/`，使用 `npm`（Makefile 会通过 `--prefix client` 调用）。
- 首次克隆后建议执行 `make init`，启用仓库自带的 Git hooks（位于 `scripts/hooks`）。

## 构建、测试与本地开发命令

- `make deps`：安装前端/后端依赖（NPM + Go）。
- `make build`：默认构建目标（前端 `js-build` + 后端 `go-build`）。
- `make lint`：运行前端 ESLint 与后端工具链（含 `gofumpt`、`govulncheck`、复杂度检查等）。
- `make test`：运行 Vitest + `go test`（默认开启 race）。
- `make js-test-e2e`：运行 Playwright E2E（用例在 `client/tests/e2e/`）。
- `make go-check`：快速后端检查（拉取工具 + lint + test）。
- `make js-typecheck`：前端 TypeScript 仅类型检查（不产出文件）。
- 文档/脚本类检查：`make md-lint`、`make sh-lint`、`make txt-lint`。
- 前端热更新：`npm --prefix client run watch:hot`。

## 本地运行与调试

- 后端二进制默认输出为 `./NullPrivate`（由 `scripts/make/go-build.sh` 控制，可用 `OUT=...` 覆盖）。
- 示例（本地前端资源）：`./NullPrivate --web-addr 0.0.0.0:3000 --local-frontend --no-check-update --verbose`。
- 运行数据/配置通常落在 `data/`、`AdGuardHome.yaml`（提交 PR 时仅包含必要的“可复现最小改动”）。

## 代码风格与命名约定

- Go：以 `gofumpt` 为准；新增文件避免下划线命名（除 `*_test.go`、`*_linux.go` 等约定）；避免引入 `scripts/make/go-lint.sh` 中被禁用的 imports。
- 前端：Prettier + ESLint（`client/.prettierrc` 采用 `tabWidth: 4`，ESLint 基于 Airbnb）。
- 前端代码中的 i18n `defaultValue` 默认使用英文；如无明确需求，不要把兜底翻译写成中文。

## 测试指南

- Go：单元测试使用 `*_test.go`，优先与被测代码同包放置，便于覆盖与重构。
- 前端：单测位于 `client/src/__tests__/`（Vitest）；端到端测试位于 `client/tests/e2e/`（Playwright）。

## 提交与 Pull Request 指南

- 提交信息遵循 Conventional Commits：`feat: ...`、`fix(scope): ...`、`refactor: ...`、`chore: ...`。
- 提交前优先跑通 `make lint` 与 `make test`；涉及前端变更建议补充 `make js-test-e2e`。
- PR 需写清动机、影响范围与验证方式；涉及 UI 变更请附截图/GIF；涉及行为变更请同步更新 `CHANGELOG.md`/相关文档。

## 安全与配置提示

- 不要在提交中包含密钥、证书私钥或真实用户数据；安全问题按 `SECURITY.md` 指引处理。
- 本地运行产生的数据建议放在 `data/` 下，并避免将生成的日志/数据库文件变更纳入 PR（除非明确需要）。
