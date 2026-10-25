# 普通用户本地运行

首次启动允许普通用户进入初始化向导。向导按实际选择的地址和端口检查绑定能力，服务启动时由操作系统执行最终的权限与占用检查。

## 构建与启动

以下命令以仓库根目录为工作目录。已有 `build/static/` 前端构建产物时，可直接构建后端；更新前端后执行 `make js-build`。

```bash
make go-build CHANNEL=development
mkdir -p data/local-dev
./NullPrivate \
  --config AdGuardHome.yaml \
  --work-dir ./data/local-dev \
  --web-addr 127.0.0.1:34020 \
  --local-frontend \
  --no-check-update
```

打开 <http://127.0.0.1:34020/>，在向导中设置：

- 管理界面：`127.0.0.1:34020`。
- DNS：`127.0.0.1:5354`，或其他可用的高位端口。
- 管理员账号和密码：通过向导自行创建。

相对配置路径以工作目录为基准，配置保存在 `data/local-dev/AdGuardHome.yaml`。保持该目录及其内容归运行用户所有且可写；如已有其他用户创建的文件，可选择新的可写工作目录。

初始化完成后，同一命令会加载已有配置。按 `Ctrl+C` 停止服务。此方式为本地前台开发运行。

## 权限与端口

- 仅绑定回环地址，使初始化界面和 DNS 供本机访问。
- 普通用户使用高位端口即可完成 Web 和 DNS 的本地开发；使用低位端口时，按操作系统要求配置对应权限。
- DNS 客户端需要支持指定端口，例如 `dig @127.0.0.1 -p 5354 example.org`；通常的系统 DNS 设置默认使用 53 端口。
- DHCP 等具有额外系统权限要求的功能，按其部署需求单独配置。

## 正向集成测试

在 Linux 普通用户、无额外 capabilities 的环境中执行：

```bash
go test -tags=integration ./internal/home \
  -run '^TestInstallUnprivileged$' -count=1 -v -timeout=5m
```

测试从当前源码构建临时二进制，使用独立临时配置和随机高位端口，验证初始化页面、端口校验、安装、登录，以及本地 DNS 重写的 UDP/TCP 查询。覆盖管理端口保持原值和切换端口两种场景，并在结束后清理测试进程。