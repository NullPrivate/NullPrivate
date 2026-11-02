---
name: nullprivate-deploy
description: "构建并发布 NullPrivate 镜像到阿里云，更新 public3 并验证 rollout、镜像 digest、HTTPS 与 DNS/DoH；适用于发布 dev、更新或回滚 public3。"
---

# NullPrivate 发布

## Linux 环境与发布范围

- 源码：`/mnt/ssd_128gb/code/nullprivate/NullPrivate`，发布包含用户要求的当前工作区修改。
- kubeconfig：`/home/jqknono/.agents/skills/jqknono-k3s/references/k3s.yaml`。
- 配置源：`/mnt/ssd_128gb/code/ecs-manager/k3s/resources/adguardprivate.com/service-traefik/public3.yaml`。
- 推送：`registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev`。
- 集群拉取：`registry-vpc.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev`。
- namespace：`adguardprivate`；Deployment：`public3`。

公网与 VPC 地址使用同一镜像存储。执行前验证路径存在；环境迁移时按用户提供的位置或当前工作区定位配置源，kubeconfig 按已安装的 `jqknono-k3s` 技能目录解析。路径仍缺失时报告具体缺项。

用户明确要求推送、部署、更新或回滚时执行对应写操作；检查请求执行只读查询。配置、Secret、证书、PVC、Service、入口资源保持现状，按用户明确要求扩大变更范围。凭据仅供工具读取，输出限于非敏感元数据。构建使用 `VERBOSE=0` 或 `1`。

## 发布前核对

1. 阅读根目录 `AGENTS.md` 和 `.vscode/tasks.json`，复用本次工作区已通过的 lint、测试与前端构建结果。
2. 核对 live 与配置源的镜像、容器名、拉取策略、节点约束，记录旧 revision、Pod 和 imageID。差异在发布前说明并处理。
3. 记录远端旧 digest：

```bash
docker buildx imagetools inspect registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev
```

4. 核对可调度节点架构。Quick 流程默认 linux/amd64；存在可调度 arm64 节点时准备对应二进制并构建多架构镜像。
5. 查询关联 CronJob、Job、initContainer 和 sidecar；仅更新本次发布范围内的镜像引用。

## 构建与推送

Go 版本与 CI 保持一致，当前使用修复标准库漏洞的 Go 1.26.10：

```bash
cd /mnt/ssd_128gb/code/nullprivate/NullPrivate
make js-build VERBOSE=0
make build-release GOTOOLCHAIN=go1.26.10 VERBOSE=0 CHANNEL=development ARCH=amd64 OS=linux SIGN=0 FRONTEND_PREBUILT=1
docker buildx build --platform linux/amd64 -f docker/Dockerfile.quick \
  -t registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev . --push
docker buildx imagetools inspect registry.cn-hangzhou.aliyuncs.com/adguardprivate/adguardprivate:dev
```

确认 release 退出码为 0，`dist/NullPrivate_linux_amd64/NullPrivate/NullPrivate` 为本次产物。已用 `--load` 构建并核对的本地镜像，可直接 `docker push` 同一标签。

记录新 index digest 以及 linux/amd64 子 manifest digest。新旧摘要相同时先核对产物与上下文，再决定是否需要 rollout。用户指定版本 tag 时使用指定 tag。

## 更新 public3

`public3` 为单副本测试环境，更新可能短暂中断。当前配置为 VPC `:dev` 和 `imagePullPolicy: Always`，同标签发布执行 restart：

```bash
kubeconfig=/home/jqknono/.agents/skills/jqknono-k3s/references/k3s.yaml
kubectl --kubeconfig "$kubeconfig" -n adguardprivate rollout restart deployment/public3
kubectl --kubeconfig "$kubeconfig" -n adguardprivate rollout status deployment/public3 --timeout=5m
```

变更 tag 时同步配置源，先 server-side dry-run，再应用对应 Deployment 变更。保留其他资源现状。

## 验收

- revision 增加，期望副本全部 Available，新 Pod Ready，重启正常。
- imageID 与新 index digest 或对应平台子 manifest digest 一致；记录具体对应关系。
- Service endpoints 指向新 Ready Pod；Gateway/HTTPRoute 状态可用。
- Pod 的 Web/DNS 启动完成；读取日志时输出经过脱敏的摘要。
- 应用内部健康探测、入口 HTTPS、DoH 均成功：

```bash
curl -sS -L --max-time 30 -o /dev/null -w 'HTTP %{http_code}; URL %{url_effective}\n' https://public3.adguardprivate.com/
curl --doh-url https://public3.adguardprivate.com/dns-query --head --silent --show-error \
  --max-time 20 https://example.com/
```

可发送 DNS wire-format 请求直接验证 DoH 响应的状态与答案，避免客户端代理绕过目标 DoH。

- 核对关联任务的镜像范围、配置源与 live tag 一致，运行 `git diff --check`。
- 公网异常时先查 origin Traefik 和入口，再查外部代理/DNS；按 `jqknono-k3s` 的故障流程定位。

## 故障与回滚

ImagePullBackOff 时读取实际拉取事件，核对 tag/digest、VPC 地址、拉取凭据引用及架构。启动失败时读取当前和 previous 日志，保留配置与数据并回滚；避免反复 restart 掩盖原因。

```bash
kubectl --kubeconfig /home/jqknono/.agents/skills/jqknono-k3s/references/k3s.yaml \
  -n adguardprivate rollout undo deployment/public3
```

默认覆盖可变 dev 标签，仅记录旧 digest。历史 revision 仍使用 dev 时，undo 可能拉到新镜像；可靠镜像回滚应使用仍可拉取的旧 digest 或不可变 tag，并同步配置源。回滚后重复完整验收。

## 交付记录

报告推送地址、新 digest、旧/新 revision、新 Pod 与实际 imageID、Ready/重启次数、endpoints、HTTPS/DoH 结果、配置源路径与修改状态，以及 dev 标签的回滚限制。部分验收失败时准确说明完成层级与剩余问题。
