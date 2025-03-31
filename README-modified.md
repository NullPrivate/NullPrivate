Modifications to the original repository:

- client(Frontend)
  - Hiding
    - Menu hidden: Encryption settings, Client settings, DHCP settings, `SETTINGS_URLS.encryption`, `SETTINGS_URLS.clients`, `SETTINGS_URLS.dhcp`
    - Hide Safe Browsing, Parental Control, `safebrowsing_enabled`, `parental_enabled`
    - Hide cache_size settings, `CACHE_CONFIG_FIELDS`
    - Hide rate limiting settings, `ratelimit_subnet_len_ipv4`
    - Hide fastest IP selection, `UPSTREAM_MODE_NAME`
    - Hide query log modification, `query_log_retention`
    - Hide statistics record modification, `statistics_retention`
    - Hide unnecessary content in setup guide, `install_devices_router_desc`
    - Hide local client queries, `local_ptr_upstreams`
  - Additions
    - Auto-fill subdomain in login interface username
    - Setup guide only shows necessary information
  - Modifications
    - Modified Logo and title
- internal(backend)
  - Additions
    - Scheduled certificate reloading
    - Rate limit support for other protocols
  - Modifications
    - Replace `github.com/AdguardTeam/dnsproxy` with `github.com/jqknono/dnsproxy`
    - Original library does not support scheduled certificate reloading, added goroutine to reload certificates every three days, `case <-time.After(3 * 24 * time.Hour):`
    - Limit upstream servers for specific domains, specific domain count limited to `255`
    - In parallel request mode, limit effective upstream DNS count to `5`
    - Limit blacklist and whitelist rules to `1 million` entries each, `dns_blocklists_desc`, `maximumCount := 1000000`
- dnsproxy
  - Original library only supports rate limiting for plain UDP requests, modified to add rate limit support for other protocols, `p.isRatelimited(ip)`
- k8s
  - Load balancer handles encryption certificate routing, DoH proxied to HTTP, DoT proxied to UDP requests
  - HTTP requests limited to 30 per second, allowing bursts of 60
  - UDP requests limited to 30 per second

## Unified Library

```shell
# github.com/AdguardTeam/dnsproxy
cp -R ../dnsproxy/fastip/ .
cp -R ../dnsproxy/internal/ .
cp -R ../dnsproxy/proxy/ .
cp -R ../dnsproxy/proxyutil/ .
cp -R ../dnsproxy/upstream/ .
```
