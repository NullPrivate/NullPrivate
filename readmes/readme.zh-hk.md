# AdGuardPrivate

AdGuardPrivate 是 _AdGuardHome_ 的分支，旨在提供具有增強功能和可定制性的 SaaS 託管版本。它託管於 [AdGuard Private](https://adguardprivate.com)。

## 主要功能

### 原有功能

1. **網絡廣告屏蔽**

   - 在您的網絡中所有設備上屏蔽廣告和追蹤器。
   - 作為 DNS 服務器運作，將追蹤域名重新路由到“黑洞”。

2. **自定義過濾規則**

   - 添加您自己的自定義過濾規則。
   - 監控和控制網絡活動。

3. **加密 DNS 支持**

   - 支持 DNS-over-HTTPS、DNS-over-TLS 和 DNSCrypt。

4. **內建 DHCP 服務器**

   - 提供開箱即用的 DHCP 服務器功能。

5. **每個客戶端配置**

   - 為個別設備配置設置。

6. **家長控制**

   - 屏蔽成人域名並在搜索引擎上強制使用安全搜索。

7. **跨平台兼容性**

   - 在 Linux、macOS、Windows 等上運行。

8. **注重隱私**
   - 除非明確配置，否則不收集使用統計數據或發送數據。

### AdGuardPrivate 新增功能

1. **帶規則列表的 DNS 路由**

   - 使用配置文件中定義的規則列表自定義 DNS 路由。
   - 支持第三方規則，如 [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)。

2. **應用特定阻擋規則列表**

   - 配置來自特定應用程序的來源阻擋。
   - 支持第三方配置以實現靈活管理。

3. **動態 DNS (DDNS)**

   - 為各種場景提供動態域名解析能力。

4. **高級速率限制**

   - 實施高效的流量管理和控制措施。

5. **增強的部署功能**
   - 支持負載平衡。
   - 自動證書維護。
   - 優化網絡連接。

詳情請訪問：[AdGuardPrivate 文檔](https://adguardprivate.com/docs/)

## 使用方法

### 下載二進制文件

您可以從 [Releases](https://github.com/AdGuardPrivate/AdGuardPrivate/releases) 頁面直接下載二進制文件。下載後，按照以下步驟運行：

```bash
./AdGuardPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### 使用 Docker 鏡像

或者，您可以使用 [Docker Hub](https://hub.docker.com/repository/docker/adguardprivate/adguardprivate) 上可用的 Docker 鏡像：

```bash
docker run --rm --name AdGuardPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf adguardprivate/adguardprivate:latest
```