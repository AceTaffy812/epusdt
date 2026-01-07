## Epusdt

## 安装须知

- 本项目修改自 https://github.com/fmnx/epusdt
- 安装过程大致与原版 epusdt 相同，但是需要替换 `static` `.env` `epusdt`。
- 数据库结构有修改，[请用这个文件](./sql/v0.0.1.sql)
- 兼容原版的 epusdt 插件（默认收 `polygon` 链，使用 `channel` 参数可同时收 `trc20` `bsc` `eth` `avax-c` `arb` 链）

**重要：本项目是自用性质，没有任何可靠性保证。请谨慎用于商业项目的收款，出现任何损失只能自己承担。**

### Etherscan API

收款需要在 .env 中填写 `etherscan_api`，不填用不了。详情请看 `.env.example` 文件

## 教程：

- 开发者接入`epusdt`文档👉🏻[开发者接入epusdt](wiki/API.md)

注意：项目中的其他教程和插件来自原版epusdt，不一定正确。

## 日志查看

持久化日志默认位于 `runtime/logs/`

为方便，收款监听的错误日志同时也会在 stdout 打印。
