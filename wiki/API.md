# 前言
开发者可通过`Epusdt`提供的`http api`将交易功能集成至任何系统

# 接口统一加密方式
### 签名算法MD5
签名生成的通用步骤如下：            

第一步，将所有非空参数值的参数按照参数名ASCII码从小到大排序（字典序），使用URL键值对的格式（即key1=value1&key2=value2…）拼接成`待加密参数`。              

重要规则：   
◆ 参数名ASCII码从小到大排序（字典序）；         
◆ 如果参数的值为空不参与签名；        
◆ 参数名区分大小写；
第二步，`待加密参数`最后拼接上`api接口认证token`得到`待签名字符串`，并对`待签名字符串`进行MD5运算，再将得到的`MD5字符串`所有字符转换为`小写`，得到签名`signature`。 注意：`signature`的长度为32个字节。

举例：

假设传送的参数如下：      
```
order_id : 20220201030210321
amount : 42
notify_url : http://example.com/notify
redirect_url : http://example.com/redirect
```

假设api接口认证token为：`epusdt_password_xasddawqe`(api接口认证token可以在`.env`文件设置)         

第一步：对参数按照key=value的格式，并按照参数名ASCII字典序排序如下：       
```
amount=42&notify_url=http://example.com/notify&order_id=20220201030210321&redirect_url=http://example.com/redirect
```
第二步：拼接API密钥并加密：
```
MD5(amount=42&notify_url=http://example.com/notify&order_id=20220201030210321&redirect_url=http://example.com/redirectepusdt_password_xasddawqe)
```

最终得到最终发送的数据：    
```
order_id : 20220201030210321
amount : 42
notify_url : http://example.com/notify
redirect_url : http://example.com/redirect
signature : 1cd4b52df5587cfb1968b0c0c6e156cd
```

### PHP加密示例
```php
    function epusdtSign(array $parameter, string $signKey)
    {
        ksort($parameter);
        reset($parameter); 
        $sign = '';
        $urls = '';
        foreach ($parameter as $key => $val) {
            if ($val == '') continue;
            if ($key != 'signature') {
                if ($sign != '') {
                    $sign .= "&";
                    $urls .= "&";
                }
                $sign .= "$key=$val"; 
                $urls .= "$key=" . urlencode($val); 
            }
        }
        $sign = md5($sign . $signKey);//密码追加进入开始MD5签名
        return $sign;
    }
```

## 接口列表

### 返回字段兼容性说明

以下说明同时适用于“创建交易”和“创建选择支付方式订单”两个接口：除 `trade_id` 和 `payment_url` 外，其他返回值不建议作为稳定接口依赖，因为其语义可能发生变化。确实需要使用其他字段时，请先确认当前 Epusdt 版本对应的接口定义。

### 1. 创建交易

创建交易订单并获取收银台链接。

#### 请求地址

```http
POST /api/v1/order/create-transaction
```

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|---|---|:---:|---|
| `order_id` | string | ✅ | 商户订单编号。 |
| `amount` | number | ✅ | 请求支付金额，当前使用 CNY，最少 `0.01`，小数点后保留 2 位。 |
| `exchange_rate` | string | ❌ | 汇率：`x` 表示 `x` 单位法币兑换 1 USDT；不填则使用系统配置汇率。 |
| `channel` | string | ❌ | 支付链与资产。不填默认为 Polygon USDT；USDC 使用 `polygon_usdc` 等复合值。完整列表见 [链与资产](ASSET.md)。 |
| `notify_url` | string | ✅ | 支付结果异步回调地址。 |
| `redirect_url` | string | ❌ | 支付成功后的同步跳转地址。 |
| `signature` | string | ✅ | 签名字符串，详见[接口统一加密方式](#接口统一加密方式)。 |

#### 请求示例

```json
{
  "order_id": "2022123321312321321",
  "amount": 100,
  "channel": "trc20",
  "notify_url": "http://example.com/",
  "redirect_url": "http://example.com/",
  "signature": "xsadaxsaxsa"
}
```
#### 响应参数

| 参数名 | 类型 | 说明 |
|---|---|---|
| `status_code` | number | 状态码，`200` 表示成功。 |
| `message` | string | 响应消息。 |
| `data.trade_id` | string | 系统交易 ID。 |
| `data.order_id` | string | 商户订单编号。 |
| `data.amount` | number | 请求支付金额（法币）。 |
| `data.actual_amount` | number | 实际需要支付的加密货币金额，币种由请求的 `channel` 决定。 |
| `data.token` | string | 收款钱包地址。 |
| `data.expiration_time` | number | 订单过期时间，Unix 时间戳（秒）。 |
| `data.payment_url` | string | 收银台订单链接。 |
| `request_id` | string | 请求 ID。 |

#### 响应示例

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "trade_id": "202203271648380592218340",
    "order_id": "9",
    "amount": 53,
    "actual_amount": 7.9104,
    "token": "trc20:TNEns8t9jbWENbStkQdVQtHMGpbsYsQjZK",
    "expiration_time": 1648381192,
    "payment_url": "http://example.com/pay/checkout-counter/202203271648380592218340"
  },
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

### 2. 创建选择支付方式订单

创建支付方式选择收银台。该接口不会立即创建链上收款订单，用户会在收银台选择支付链与资产后再创建实际交易。请求中的 `channel` 字段会被忽略。

#### 请求地址

```http
POST /api/v1/order/create-order
```

为兼容某些会自动向接口地址附加路径的插件，以下形式也会按本接口处理：

```
POST /api/v1/order/create-order/api/v1/order/xxxx
```

#### 请求参数

请求字段与[创建交易](#1-创建交易)完全一致，其中 `channel` 可传可不传，但不会影响最终的支付链和币种；最终支付方式由付款用户在收银台选择。

#### 请求示例

```json
{
  "order_id": "2022123321312321321",
  "amount": 100,
  "channel": "trc20",
  "notify_url": "http://example.com/",
  "redirect_url": "http://example.com/",
  "signature": "xsadaxsaxsa"
}
```

#### 响应参数

| 参数名 | 类型 | 说明 |
|---|---|---|
| `status_code` | number | 状态码，`200` 表示成功。 |
| `message` | string | 响应消息。 |
| `data.fiat` | string | 交易法币类型，当前固定为 `CNY`。 |
| `data.trade_id` | string | 系统交易 ID，同时也是最终写入订单的交易 ID。 |
| `data.order_id` | string | 商户订单编号。 |
| `data.amount` | string | 请求支付金额（法币）。 |
| `data.status` | string | 订单状态，`1` 表示待付款。 |
| `data.expiration_time` | number | 订单有效期（秒）。 |
| `data.payment_url` | string | 收银台订单链接。 |
| `request_id` | string | 请求 ID。 |

#### 响应示例

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "fiat": "CNY",
    "trade_id": "202610051648208648961728",
    "order_id": "2022123321312321321",
    "amount": "100.00",
    "status": "1",
    "expiration_time": 600,
    "payment_url": "http://example.com/pay/checkout-order/202610051648208648961728"
  },
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

# 异步回调

支付成功后，`Epusdt`会向目标服务器发生异步通知，告知该笔交易已经支付完成。          
失败`Epusdt`最高最多重试5次，请注意验证消息签名。      
目标服务器处理完成后请返回字符串`ok`即可，否则`Epusdt`会一直重试发送消息，最高5次     

POST 【异步回调地址】

> Body 请求参数

```json
{
  "trade_id": "202203251648208648961728",
  "order_id": "2022123321312321321",
  "amount": 100,
  "actual_amount": 15.625,
  "token": "trc20:TNEns8t9jbWENbStkQdVQtHMGpbsYsQjZK",
  "block_transaction_id": "123333333321232132131",
  "signature": "xsadaxsaxsa",
  "status": 2
}
```

### 请求参数

|名称|位置| 类型     |必选| 中文名                 | 说明              |
|---|---|--------|---|---------------------|-----------------|
|body|body| object | 否 ||                     |
|» trade_id|body| string | 是 | 交易号                 |                 |
|» order_id|body| string | 是 | 请求支付订单号             |                 |
|» amount|body| float  | 是 | 支付金额(CNY)           | 小数点保留后2位 |
|» actual_amount|body| float  | 是 | 实际需要支付的稳定币金额 | 币种由原请求的 `channel` 确定；回调字段及签名保持不变 |
|» token|body| string | 是 | 钱包地址                | |
|» block_transaction_id|body| string | 是 | 区块交易号               |  |
|» signature|body| string | 是 | 签名                  |                 |
|» status|body| int    | 是 | 订单状态                | 1：等待支付，2：支付成功，3：已过期        | 

# status_code返回状态码及含义

| 状态码 | 说明  | 
|-----|-----|
|400|系统错误|
|401|签名认证错误|
|10002|支付交易已存在，请勿重复创建|
|10003|无可用钱包地址，无法发起支付|
|10004|支付金额有误, 无法满足最小支付单位|
|10005|无可用金额通道|
|10006|汇率计算错误|
|10007|订单区块已处理|
|10008|订单不存在|
|10009|无法解析参数|
