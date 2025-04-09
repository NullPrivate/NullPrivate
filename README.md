# AdGuardPrivate

> Available in other languages: [العربية](./readmes/readme.ar-sa.md), [Deutsch](./readmes/readme.de-de.md), [Español](./readmes/readme.es-es.md), [Français (Canada)](./readmes/readme.fr-ca.md), [Français (France)](./readmes/readme.fr-fr.md), [日本語](./readmes/readme.ja-jp.md), [한국어](./readmes/readme.ko-kr.md), [Português (Brasil)](./readmes/readme.pt-br.md), [Русский](./readmes/readme.ru-ru.md), [简体中文](./readmes/readme.zh-cn.md), [繁體中文 (香港)](./readmes/readme.zh-hk.md)

AdGuardPrivate is a fork of _AdGuardHome_, designed to provide a SaaS-hosted version with enhanced features and customizability. It is hosted on [AdGuard Private](https://adguardprivate.com).

## Key Features

### Original Features

1. **Network-Wide Ad Blocking**

   - Blocks ads and trackers across all devices in your network.
   - Operates as a DNS server that re-routes tracking domains to a “black hole.”

2. **Custom Filtering Rules**

   - Add your own custom filtering rules.
   - Monitor and control network activity.

3. **Encrypted DNS Support**

   - Supports DNS-over-HTTPS, DNS-over-TLS, and DNSCrypt.

4. **Built-in DHCP Server**

   - Provides DHCP server functionality out-of-the-box.

5. **Per-Client Configuration**

   - Configure settings for individual devices.

6. **Parental Control**

   - Blocks adult domains and enforces Safe Search on search engines.

7. **Cross-Platform Compatibility**

   - Runs on Linux, macOS, Windows, and more.

8. **Privacy-Focused**
   - Does not collect usage statistics or send data unless explicitly configured.

### New Features by AdGuardPrivate

1. **DNS Routing with Rule Lists**

   - Customize DNS routing using rule lists defined in the configuration file.
   - Supports third-party rules like [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat).

2. **Application-Specific Blocking Rule Lists**

   - Configure blocking of sources from specific applications.
   - Supports third-party configurations for flexible management.

3. **Dynamic DNS (DDNS)**

   - Provides dynamic domain name resolution capabilities for various scenarios.

4. **Advanced Rate Limiting**

   - Implements efficient traffic management and control measures.

5. **Enhanced Deployment Features**
   - Load balancing support.
   - Automatic certificate maintenance.
   - Optimized network connections.

For detailed documentation, visit: [AdGuardPrivate Documentation](https://adguardprivate.com/docs/)

## How to Use

### Download Binary

You can download the binary directly from the [Releases](https://github.com/AdGuardPrivate/AdGuardPrivate/releases) page. Once downloaded, follow these steps to run it:

```bash
./AdGuardPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### Use Docker Image

Alternatively, you can use the Docker image available on [Docker Hub](https://hub.docker.com/repository/docker/adguardprivate/adguardprivate):

```bash
docker run --rm --name AdGuardPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf adguardprivate/adguardprivate:latest
```
