## Epusdt

## 项目介绍

- 本项目修改自 https://github.com/fmnx/epusdt
- 安装过程大致与原版 epusdt 相同，但是需要替换 `static` `.env` `epusdt`。
- 兼容原版的 epusdt 插件（使用原版接口默认收 `polygon` 链，使用 `channel` 参数可同时收其他链）
- 兼容“类 bepusdt”的 `/api/v1/order/create-order` 接口，实现用户到收银台自选币种和网络。

**❗️作者声明：**

1. 本项目是自用性质，没有任何可靠性保证。请谨慎使用，出现任何损失只能自己承担。
2. 本项目仅开源，不接受 Issue 和 Pull Request。无任何技术支持。
3. 本项目为研究学习区块链的开源项目，不提供任何形式的收费服务 (谨防诈骗) ，不鼓励任何衍生金融属性的交易行为，不负责任何使用本项目进行的三方行为！

### Etherscan API

EVM 链收款需要在 .env 中填写 `etherscan_api`，不填用不了。详情请看 `.env.example` 文件

## 教程：

- 数据库安装与更新说明请参考👉🏻[数据库文档](./sql/README.md)
- 开发者接入`epusdt`文档👉🏻[API文档](wiki/API.md)
- 链与资产说明请参考👉🏻[链与资产文档](wiki/ASSET.md)
- 环境变量配置说明请参考👉🏻[环境变量文档](wiki/ENV.md)

## 日志查看

持久化日志默认位于 `runtime/logs/`

为方便，收款监听的错误日志同时也会在 stdout 打印。
