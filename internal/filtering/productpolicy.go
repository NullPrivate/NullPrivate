package filtering

// SafeBrowsingAndParentalEnabled 是安全浏览与家长控制的固定产品策略。
// 两项功能保留配置、接口和历史数据兼容实现，运行时统一停用；普通规则、
// 安全搜索与防沉迷沿用各自配置。使用常量保证旧配置和客户端设置均遵循此策略。
const SafeBrowsingAndParentalEnabled = false
