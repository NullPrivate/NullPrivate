# AdGuardPrivate

AdGuardPrivate 是 _AdGuardHome_ 的一个分支，旨在提供一个具有增强功能和可定制性的 SaaS 托管版本。它托管在 [AdGuard Private](https://adguardprivate.com)。

## 主要功能

### 原始功能

1. **网络范围广告屏蔽**

   - 在您的网络中跨所有设备屏蔽广告和跟踪器。
   - 作为一个 DNS 服务器运行，将跟踪域名重新路由到“黑洞”。

2. **自定义过滤规则**

   - 添加您自己的自定义过滤规则。
   - 监控和控制网络活动。

3. **加密 DNS 支持**

   - 支持 DNS-over-HTTPS、DNS-over-TLS 和 DNSCrypt。

4. **内置 DHCP 服务器**

   - 开箱即用提供 DHCP 服务器功能。

5. **每客户端配置**

   - 为单个设备配置设置。

6. **家长控制**

   - 屏蔽成人域名并在搜索引擎上强制启用安全搜索。

7. **跨平台兼容性**

   - 在 Linux、macOS、Windows 等系统上运行。

8. **注重隐私**
   - 除非明确配置，否则不收集使用统计数据或发送数据。

### AdGuardPrivate 新增功能

1. **使用规则列表的 DNS 路由**

   - 使用配置文件中定义的规则列表自定义 DNS 路由。
   - 支持第三方规则，如 [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)。

2. **特定应用的屏蔽规则列表**

   - 配置从特定应用程序的源进行屏蔽。
   - 支持第三方配置以实现灵活管理。

3. **动态 DNS (DDNS)**

   - 为各种场景提供动态域名解析能力。

4. **高级速率限制**

   - 实施高效的流量管理和控制措施。

5. **增强的部署功能**
   - 支持负载均衡。
   - 自动证书维护。
   - 优化网络连接。

有关详细文档，请访问：[AdGuardPrivate 文档](https://adguardprivate.com/docs/)

## 使用方法

### 下载二进制文件

您可以从 [Releases](https://github.com/AdGuardPrivate/AdGuardPrivate/releases) 页面直接下载二进制文件。下载后，按照以下步骤运行：

```bash
./AdGuardPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### 使用 Docker 镜像

或者，您可以使用 [Docker Hub](https://hub.docker.com/repository/docker/adguardprivate/adguardprivate) 上可用的 Docker 镜像：

```bash
docker run --rm --name AdGuardPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf adguardprivate/adguardprivate:latest
```