# 环境变量说明

配置文件为 [`src/.env.example`](../src/.env.example)。复制为运行目录的 `.env` 后，按部署环境填写。

| 配置 | 必填 | 说明 |
|---|---|---|
| `app_uri` | 是 | 对外访问地址，用于生成收银台链接。 |
| `etherscan_api` | EVM 链需要 | Etherscan V2 API Key，用于 EVM 代币转账监听。 |
| `mysql_*` | 是 | MySQL 连接参数。 |
| `redis_*` | 是 | Redis 与异步锁/队列连接参数。 |
| `api_auth_token` | 是 | API 请求与回调签名密钥。 |
| `order_expiration_time` | 否 | 订单过期分钟数，默认 `10`。 |
| `forced_usdt_rate` | 否 | 强制支付汇率；USDT、USDC 均使用该稳定币汇率逻辑。 |
| `enabled_channel` | 否 | 启用网络/资产列表，以逗号分隔；支持 `chain_asset`（如 `polygon_usdc,trc20_usdt`）或基础网络名（如 `polygon`）。列表顺序同时决定收银台的网络显示顺序，同一网络取首次出现的位置；留空表示启用全部内置资产并使用内置顺序。 |

可用链、资产、复合通道值请查阅 [`wiki/ASSET.md`](ASSET.md)。钱包表的 `channel` 始终填写基础链名（如 `polygon`），不可填写 `polygon_usdc`。
