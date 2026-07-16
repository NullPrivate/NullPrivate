---
name: nullprivate-deploy
description: "构建并发布 NullPrivate 镜像到阿里云容器镜像服务，更新 k3s 工作负载并完成 rollout、镜像 digest、Pod、Service、Gateway/Ingress、HTTPS 与 DNS/DoH 验收。Use when: 推送 dev 镜像、发布 NullPrivate、更新 public3 Pod、部署 adguardprivate、回滚 public3、检查 NullPrivate 发布。"
argument-hint: "例如：推送 dev 镜像并更新 public3；发布指定版本；检查或回滚 public3"
user-invocable: true
disable-model-invocation: false
---

# NullPrivate Deploy

为当前仓库执行可验证、可回滚的镜像发布。默认目标是：

- 源码：当前 `NullPrivate` 工作区。
- 推送镜像：`registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:<tag>`。
- 集群拉取镜像：`registry-vpc.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:<tag>`。
- kubeconfig：`C:\Users\Administrator\.agents\skills\jqknono-k3s\references\k3s.yaml`。
- namespace：`adguardprivate`。
- dev 工作负载：`deployment/public3`。
- source of truth：`F:\code\ecs-manager\k3s\resources\adguardprivate.com\service-traefik\public3.yaml`。

公网 registry 与 VPC registry 指向同一份阿里云镜像存储：本地只向公网地址 push，集群通过 VPC 地址 pull。

此 skill 面向当前固定 Windows 运维环境，保留上述绝对路径。执行前检查路径存在；路径失效时停止并报告，不自动搜索或复制 kubeconfig。默认直接覆盖可变的 `:dev` 标签，不额外创建备份标签。

## When to Use

- 用户要求推送 `dev`、版本镜像或发布 `NullPrivate`。
- 用户要求更新、重启、验证或回滚 `public3` Pod。
- 排查发布后的 `ImagePullBackOff`、rollout 超时、live digest 不一致。

不要用于 `null_private` 后台的 `adguardprivate_master` 镜像；那是另一个仓库和工作负载。

## Safety Rules

- 发布是外部写操作。只有用户明确要求推送、部署、更新 Pod 或回滚时才执行变更；“检查”“看看”默认只读。
- 不提交、不打印、不复制 registry 凭据、kubeconfig 证书、Token 或其他环境变量。
- 构建命令使用 `VERBOSE=0` 或 `VERBOSE=1`。禁止使用 `VERBOSE=2`，因为仓库脚本会输出整个环境。
- 不修改 `AdGuardHome.yaml`、Secret、证书、PVC、Service、Gateway 或 Ingress，除非用户明确要求。
- 不使用 `git reset --hard`、`git checkout --` 等方式清理工作区；镜像应包含用户当前明确要求发布的未提交修改。
- 不因 tag 相同就假设 Pod 已更新。必须核对 registry digest 与 Pod `imageID`。
- rollout 前记录旧 revision、Pod、imageID；失败时保留现场并给出回滚命令。
- 覆盖 `:dev` 前只记录旧 digest，不创建 `dev-backup-*` 或其他备份标签；最终报告必须说明这会限制镜像级回滚能力。
- `public3` 是单副本测试环境，更新期间可能短暂中断；不要同时更新其他 Deployment。

## Procedure

### 1. Preflight

1. 读取仓库根目录 `AGENTS.md` 和 `.vscode/tasks.json`，确认当前约定没有变化。
2. 运行已存在的聚焦测试和前端生产构建。涉及过滤模块和前端时，至少执行：

   ```powershell
   Set-Location 'F:\code\nullprivate\NullPrivate'
   go test ./internal/filtering
   make js-build VERBOSE=1
   ```

3. 查询集群，不做变更：

   ```powershell
   $kubeconfig = 'C:\Users\Administrator\.agents\skills\jqknono-k3s\references\k3s.yaml'
   kubectl --kubeconfig $kubeconfig config current-context
   kubectl --kubeconfig $kubeconfig get deployment,pod,service -n adguardprivate -l app=public3 -o wide
   ```

4. 从 live Deployment 读取容器名、镜像、`imagePullPolicy`、revision；从最新 Pod 记录旧 `imageID`。不要猜容器名或 registry。
5. 检查 source of truth，确认镜像仍为 VPC 地址和目标 tag。若 live 与 YAML 不一致，先报告差异；需要改 YAML 时同步修改并验证，避免以后被覆盖。
6. 用 `docker buildx imagetools inspect` 记录推送前远端 digest。
7. 检查目标节点架构。默认 Quick dev 流程只构建 `linux/amd64`；若目标 Pod可能调度到 arm64，改用多架构发布，不能继续单架构镜像。

### 2. Build Frontend and Linux Binary

默认 dev/public3 快速发布走仓库 Quick 流程：

```powershell
Set-Location 'F:\code\nullprivate\NullPrivate'
make js-build VERBOSE=1
make build-release VERBOSE=1 CHANNEL='development' ARCH='amd64' OS='linux' SIGN='0' FRONTEND_PREBUILT='1'
```

必须确认：

- `dist\NullPrivate_linux_amd64\NullPrivate\NullPrivate` 的修改时间是本次构建时间。
- 命令退出码为 0。
- 不能只看 VS Code 复合任务的第一段输出；必须确认 release 和 push 阶段分别完成。

#### Windows 缺少 zip 时

`build-release.sh` 会在任何平台构建前统一检查 `zip`，即使 Linux amd64 最终只生成 `.tar.gz`。若仅因 `pieces don't fit, 'zip' not found` 失败，且 Quick Dockerfile 只需要 Linux 二进制，可以使用同一底层构建脚本生成精确产物：

```powershell
Set-Location 'F:\code\nullprivate\NullPrivate'
New-Item -ItemType Directory -Force '.\dist\NullPrivate_linux_amd64\NullPrivate' | Out-Null
$env:GOOS = 'linux'
$env:GOARCH = 'amd64'
$env:GOARM = ''
$env:GOMIPS = ''
$env:CHANNEL = 'development'
$env:VERSION = 'v0.0.0'
$env:RACE = '0'
$env:NEXTAPI = '0'
$env:OUT = './dist/NullPrivate_linux_amd64/NullPrivate/NullPrivate'
$env:VERBOSE = '1'
& 'C:\Program Files\Git\bin\sh.exe' './scripts/make/go-build.sh'
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Get-Item $env:OUT | Select-Object FullName, Length, LastWriteTime
```

只允许在以下条件同时满足时走该 fallback：

- 前端 `make js-build` 已成功，仓库根目录 `build/` 已更新。
- 失败原因仅为缺少 `zip`，不是 Go 编译、测试或前端构建错误。
- 目标只需要 Quick Dockerfile 中的 Linux 二进制，不要求完整 release 归档和 checksums。

### 3. Build and Push Image

默认 dev/public3：

```powershell
Set-Location 'F:\code\nullprivate\NullPrivate'
docker buildx build `
  --platform linux/amd64 `
  -f docker/Dockerfile.quick `
  -t registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev `
  . --push
```

推送后执行：

```powershell
docker buildx imagetools inspect registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev
```

记录新的顶层 manifest digest。新 digest 必须与推送前不同；若相同，停止 rollout 并调查是否使用了旧产物或构建上下文。

版本发布时使用用户明确给出的 tag，不自动覆盖 `latest`。需要多架构时使用仓库完整发布流程或分别准备 amd64/arm64 产物，不能把单架构二进制标记为多架构镜像。

### 4. Roll Out public3

`public3.yaml` 使用 VPC 镜像和 `imagePullPolicy: Always`。当 tag 仍为 `dev` 时，使用 restart 生成新的 Pod template revision：

```powershell
$kubeconfig = 'C:\Users\Administrator\.agents\skills\jqknono-k3s\references\k3s.yaml'
kubectl --kubeconfig $kubeconfig rollout restart deployment/public3 -n adguardprivate
kubectl --kubeconfig $kubeconfig rollout status deployment/public3 -n adguardprivate --timeout=5m
```

若发布的是新 tag：

1. 先修改 `F:\code\ecs-manager\k3s\resources\adguardprivate.com\service-traefik\public3.yaml` 的镜像 tag。
2. 运行 server-side dry-run。
3. apply source of truth。
4. 等待 rollout。

不要只运行 `kubectl set image` 而不更新 YAML，否则后续 apply 会回退 live 配置。

### 5. Validate

只有以下检查都通过，才能报告“部署完成”：

1. Deployment revision 增加，`availableReplicas` 等于期望副本数。
2. 只保留新 Pod；状态为 `Running`、`Ready=true`、重启次数无异常。
3. Pod `imageID` 的 digest 等于本次 push 的新 manifest digest。公网与 VPC registry 前缀不同是正常的。
4. `service/public3` endpoints 非空，并指向新 Ready Pod。
5. Gateway/HTTPRoute 或当前入口资源处于可用状态。
6. 新 Pod 日志显示 Web 与 DNS 服务启动完成；检查错误但不要回显敏感配置。
7. 公网页面可访问：

   ```powershell
   Invoke-WebRequest -Uri 'https://public3.adguardprivate.com' -UseBasicParsing -MaximumRedirection 5
   ```

   期望最终到达登录页并返回 HTTP 200。

8. DoH 链路可用：

   ```powershell
   curl.exe --doh-url 'https://public3.adguardprivate.com/dns-query' `
     --head --silent --show-error --max-time 20 `
     --write-out "HTTP %{http_code}; remote_ip=%{remote_ip}`n" `
     'https://example.com/'
   ```

9. 查询 `CronJob`、Job、initContainer 和 sidecar 是否引用同一镜像。若存在，必须同步或明确说明为什么不需要；当前 `public3` 通常没有活动 CronJob，但每次都要实查。
10. `git diff --check` 无空白错误；source of truth 与 live image tag 一致。

日志中已有但不阻断启动的 warning 需单独报告，例如上游 HTTP/3 超时后回落到 HTTP/2；不要把它误判成 rollout 失败。核心 DNS 查询失败、CrashLoop、空 endpoints 或 digest 不一致则不能宣称完成。

## Failure Branches

### ImagePullBackOff

1. `kubectl describe pod` 读取真实拉取错误。
2. 确认公网 registry 的目标 tag 和 digest 已存在。
3. 确认 live image 使用 VPC registry。
4. 检查 namespace 的 `imagePullSecrets` 和 Deployment 引用。
5. 检查节点架构是否包含在 manifest 中。
6. 修复后重新 rollout；不要反复 restart 掩盖根因。

### Rollout Timeout or CrashLoopBackOff

1. 查看新 Pod 的 `logs --previous`、当前日志和 `describe` 事件。
2. 保留旧 revision 信息；不要删除 PVC、Secret 或配置文件。
3. 若新镜像无法启动，执行回滚并等待回滚完成。

### Registry Digest Updated but Pod Digest Old

1. 确认 `imagePullPolicy: Always`。
2. 确认 rollout 确实创建了新 Pod 和新 revision。
3. 比较 Pod `imageID`，不要比较镜像 tag 字符串。
4. 检查 VPC registry 是否已同步到同一 digest；必要时短暂重查 registry，但不要无限轮询。

## Rollback

默认回滚到上一 revision：

```powershell
$kubeconfig = 'C:\Users\Administrator\.agents\skills\jqknono-k3s\references\k3s.yaml'
kubectl --kubeconfig $kubeconfig rollout undo deployment/public3 -n adguardprivate
kubectl --kubeconfig $kubeconfig rollout status deployment/public3 -n adguardprivate --timeout=5m
```

回滚后重复 Pod、digest、endpoints、HTTPS 和 DoH 验收。注意：Deployment 历史若仍引用可变 `:dev` tag，`rollout undo` 可能再次拉取当前 dev digest，不能保证恢复旧镜像内容。当前默认不创建备份标签；需要可重复回滚时，应改为发布不可变版本 tag 或 digest，并在 source of truth 中记录。最终报告必须明确这个限制。

## Completion Report

最终报告至少包含：

- 推送地址、tag 和新 digest。
- Deployment、namespace、旧/新 revision。
- 新 Pod 名、Ready 状态、重启次数和实际 `imageID`。
- Service endpoints、入口 HTTP 状态和 DoH 验证结果。
- source of truth 是否修改以及路径。
- 构建是否使用 `build-release` 或缺少 `zip` 时的底层 fallback。
- 发现的非阻断 warning。
- 可执行的回滚命令，以及可变 `dev` tag 对回滚可靠性的限制。
