package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/mq"
	"github.com/assimon/luuu/mq/handle"
	"github.com/assimon/luuu/telegram"
	"github.com/assimon/luuu/util/http_client"
	"github.com/assimon/luuu/util/log"
	"github.com/golang-module/carbon/v2"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
)

const (
	AptosGraphqlUrl = "https://api.mainnet.aptoslabs.com/v1/graphql"
)

type aptosGraphqlResp struct {
	Data struct {
		AccountTransactions []struct {
			TransactionVersion      int64 `json:"transaction_version"`
			FungibleAssetActivities []struct {
				Amount               int64  `json:"amount"`
				AssetType            string `json:"asset_type"`
				IsTransactionSuccess bool   `json:"is_transaction_success"`
				Type                 string `json:"type"`
				OwnerAddress         string `json:"owner_address"`
			} `json:"fungible_asset_activities"`
			UserTransaction struct {
				Timestamp string `json:"timestamp"`
			} `json:"user_transaction"`
		} `json:"account_transactions"`
	} `json:"data"`
}

func AptosApiScan(asset model.PaymentAsset, token string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("AptosCallBack:", time.Now().UTC().Format("2006-01-02 15:04:05 MST"), err)
			log.Sugar.Error(err)
		}
	}()

	if !data.IsWalletLocked(asset.Chain, asset.Symbol, token) {
		return
	}

	client := http_client.GetHttpClient()

	// 构造 GraphQL 请求体
	payload := map[string]interface{}{
		"query": `query AccountTransactionsData($address: String, $limit: Int, $offset: Int) {
  account_transactions(
    where: {account_address: {_eq: $address}}
    order_by: {transaction_version: desc}
    limit: $limit
    offset: $offset
  ) {
    transaction_version
    fungible_asset_activities {
      amount
      asset_type
      is_transaction_success
      type
      owner_address
    }
    user_transaction {
      timestamp
    }
  }
}`,
		"variables": map[string]interface{}{
			"address": token,
			"limit":   25,
			"offset":  0,
		},
		"operationName": "AccountTransactionsData",
	}

	bodyBytes, _ := json.Marshal(payload)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bodyBytes).
		Post(AptosGraphqlUrl)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != http.StatusOK {
		panic(resp.StatusCode())
	}

	var gqlResp aptosGraphqlResp
	err = json.Unmarshal(resp.Body(), &gqlResp)
	if err != nil {
		panic(err)
	}

	if len(gqlResp.Data.AccountTransactions) == 0 {
		return
	}

	decimalDivisor := decimal.New(1, int32(asset.Decimals))

	// 逐条交易检查
	for _, tx := range gqlResp.Data.AccountTransactions {
		var txTimestampMillis int64
		if tx.UserTransaction.Timestamp != "" {
			// Aptos 返回没有时区信息，按 UTC 解析（如果你的数据带时区，请使用 RFC3339Nano）
			// 兼容微秒长度（最多 6 位）：
			parsed, perr := time.Parse("2006-01-02T15:04:05.999999", tx.UserTransaction.Timestamp)
			if perr != nil {
				// 作为兜底尝试 RFC3339Nano
				parsed, perr = time.Parse(time.RFC3339Nano, tx.UserTransaction.Timestamp)
				if perr != nil {
					// 如果解析失败，跳过此条交易
					log.Sugar.Warnf("AptosCallBack: failed to parse timestamp %s: %v", tx.UserTransaction.Timestamp, perr)
					continue
				}
			}
			txTimestampMillis = parsed.UTC().UnixNano() / int64(time.Millisecond)
		} else {
			// 没时间戳则跳过
			continue
		}

		for _, act := range tx.FungibleAssetActivities {
			// 只关心成功的 deposit，并且 asset_type 匹配，owner_address 为目标地址（即收到方）
			if !act.IsTransactionSuccess {
				continue
			}
			if !strings.EqualFold(act.AssetType, asset.Contract) {
				continue
			}
			if !strings.Contains(strings.ToLower(act.Type), "::deposit") {
				continue
			}
			if !strings.EqualFold(act.OwnerAddress, token) {
				continue
			}

			// 计算实际金额（注意：根据代币 decimals 调整 decimalDivisor）
			amountDecimal := decimal.NewFromInt(act.Amount).Div(decimalDivisor)
			amount := amountDecimal.InexactFloat64()

			// 根据 钱包地址 + amount 查找 tradeId（沿用你现有逻辑）
			tradeId, err := data.GetTradeIdByWalletAddressAndAmount(asset.Chain, asset.Symbol, token, amount)
			if err != nil {
				panic(err)
			}
			if tradeId == "" {
				// 没找到订单，继续下一个活动
				continue
			}

			order, err := data.GetOrderInfoByTradeId(tradeId)
			if err != nil {
				panic(err)
			}

			// 区块/交易的确认时间必须在订单创建时间之后
			createTime := order.CreatedAt.TimestampWithMillisecond()
			if txTimestampMillis < createTime {
				warnOrderCannotActuallyBeMatchedOnce(
					asset.Chain,
					tradeId,
					fmt.Sprintf("aptos_tx_version:%d", tx.TransactionVersion),
				)
				continue
			}

			// 调用订单处理（沿用你的 request 结构）
			req := &request.OrderProcessingRequest{
				Address: token,
				TradeId: tradeId,
				Amount:  amount,
				// 使用 transaction_version 作为区块/交易 id 表示
				BlockTransactionId: fmt.Sprintf("%d", tx.TransactionVersion),
				Chain:              asset.Chain,
				Asset:              asset.Symbol,
			}
			err = OrderProcessing(req)
			if err != nil {
				panic(err)
			}

			// 回调队列
			orderCallbackQueue, _ := handle.NewOrderCallbackQueue(order)
			mq.MClient.Enqueue(orderCallbackQueue, asynq.MaxRetry(5))

			// 发送机器人消息（格式可按需调整）
			msgTpl := `
<b>📢📢 有新的 Aptos 交易支付成功！</b>
<pre>收款交易类型：%s</pre>
<pre>交易号：%s</pre>
<pre>订单号：%s</pre>
<pre>请求支付金额：%f cny</pre>
<pre>实际支付金额：%f %s</pre>
<pre>钱包地址：%s</pre>
<pre>订单创建时间：%s</pre>
<pre>支付成功时间：%s</pre>
<pre>aptos_tx_version: %d</pre>
`
			msg := fmt.Sprintf(msgTpl,
				asset.Chain+"_"+asset.Symbol,
				order.TradeId,
				order.OrderId,
				order.Amount,
				order.ActualAmount,
				strings.ToUpper(asset.Symbol),
				order.WalletAddress,
				order.CreatedAt.ToDateTimeString(),
				carbon.Now().ToDateTimeString(),
				tx.TransactionVersion,
			)
			telegram.SendToBot(msg)
		}
	}
}
