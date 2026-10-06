# 链、资产与 API 使用情况

## 支持的链与 API 使用情况

| 链 | 转账扫描 API | API Key |
| --- | --- | --- |
| `trc20` | Tronscan | 暂时无需 API Key |
| `aptos` | Aptoslabs | 暂时无需 API Key |
| `bsc` | Etherscan | 必须填写 API Key（必须付费账号） |
| `eth` | Etherscan | 必须填写 API Key（免费账号可用） |
| `avax-c` | Etherscan | 必须填写 API Key（必须付费账号） |
| `plasma` | Etherscan | 必须填写 API Key（免费账号可用） |
| `polygon` | Etherscan | 必须填写 API Key（免费账号可用） |

## 支持的资产

创建订单通过既有 `channel` 参数选择资产。未提供 `channel` 时为 `polygon` 的 USDT；所有历史基础链值仍表示 USDT。复合通道使用 `链名_币种` 格式，例如 `polygon_usdc`。

| 请求 `channel` | 网络 | 资产 | 合约/Metadata 地址 | 精度 |
|---|---|---|---|---:|
| `trc20` | TRON | USDT | `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t` | 6 |
| `polygon` / `polygon_usdt` | Polygon | USDT | `0xc2132d05d31c914a87c6611c10748aeb04b58e8f` | 6 |
| `polygon_usdc` | Polygon | USDC | `0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359` | 6 |
| `bsc` / `bsc_usdt` | BNB Smart Chain | USDT | `0x55d398326f99059fF775485246999027B3197955` | 18 |
| `bsc_usdc` | BNB Smart Chain | USDC | `0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d` | 18 |
| `avax-c` / `avax-c_usdt` | Avalanche C-Chain | USDT | `0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7` | 6 |
| `avax-c_usdc` | Avalanche C-Chain | USDC | `0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E` | 6 |
| `eth` / `eth_usdt` | Ethereum | USDT | `0xdac17f958d2ee523a2206206994597c13d831ec7` | 6 |
| `eth_usdc` | Ethereum | USDC | `0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48` | 6 |
| `plasma` / `plasma_usdt` | Plasma | USDT | `0xB8CE59FC3717ada4C02eaDF9682A9e934F625ebb` | 6 |
| `plasma_usdc` | Plasma | USDC | `0x2d661C89D812261039AF9764eceaAee884f5F67F` | 6 |
| `aptos` / `aptos_usdt` | Aptos | USDT | `0x357b0b74bc833e95a115ad22604854d6b0fca151cecd94111770e5d6ffc9dc2b` | 6 |
| `aptos_usdc` | Aptos | USDC | `0xbae207659db88bea0cbead6da0ed00aac12edcdda169e591cd41c94180b46f3b` | 6 |

TRC20 仅支持 USDT，不存在 `trc20_usdc`。更多配置说明请见 [`wiki/ENV.md`](ENV.md)。
