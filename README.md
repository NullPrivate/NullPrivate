This project is forked from AdGuardHome and provides a SaaS hosted version at [adguardprivate.com](https://adguardprivate.com). Compared to the original version, it adds more customizable features, including:

1. **DNS traffic splitting based on rule lists**  
   Rules sourced from [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat). The default rule is `gfwlist`, which can be customized in the configuration file.

2. **Support for blocking specified application configuration sources**  
   Can block third-party configuration sources for applications, providing more flexible management capabilities.

3. **DDNS functionality**  
   Dynamic domain name resolution capabilities to meet more use case requirements.

4. **More advanced rate limiting measures**  
   Provides more efficient traffic management and control.

In addition to one-click deployment, AdGuardPrivate also offers the following enhanced features:

- **Load balancing support**
- **Automatic certificate maintenance**
- **Optimized network connections**

For more details, please refer to: [AdGuardPrivate Documentation](https://adguardprivate.com/docs/)
