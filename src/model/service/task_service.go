package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/mq"
	"github.com/assimon/luuu/mq/handle"
	"github.com/assimon/luuu/telegram"
	"github.com/assimon/luuu/util/http_client"
	"github.com/assimon/luuu/util/json"
	"github.com/assimon/luuu/util/log"
	"github.com/golang-module/carbon/v2"
	"github.com/gookit/goutil/stdutil"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
)

const UsdtTrc20ApiUri = "https://apilist.tronscanapi.com/api/transfer/trc20"
const EtherscanApiUri = "https://api.etherscan.io/v2/api"

type UsdtTrc20Resp struct {
	PageSize int         `json:"page_size"`
	Code     int         `json:"code"`
	Data     []Trc20Data `json:"data"`
}

type EtherscanResp struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Data    []EtherscanResult `json:"result"`
}

type TokenInfo struct {
	TokenID      string `json:"tokenId"`
	TokenAbbr    string `json:"tokenAbbr"`
	TokenName    string `json:"tokenName"`
	TokenDecimal int    `json:"tokenDecimal"`
	TokenCanShow int    `json:"tokenCanShow"`
	TokenType    string `json:"tokenType"`
	TokenLogo    string `json:"tokenLogo"`
	TokenLevel   string `json:"tokenLevel"`
	IssuerAddr   string `json:"issuerAddr"`
	Vip          bool   `json:"vip"`
}

type Trc20Data struct {
	Amount         string `json:"amount"`
	ApprovalAmount string `json:"approval_amount"`
	BlockTimestamp int64  `json:"block_timestamp"`
	Block          int    `json:"block"`
	From           string `json:"from"`
	To             string `json:"to"`
	Hash           string `json:"hash"`
	Confirmed      int    `json:"confirmed"`
	ContractType   string `json:"contract_type"`
	ContracTType   int    `json:"contractType"`
	Revert         int    `json:"revert"`
	ContractRet    string `json:"contract_ret"`
	EventType      string `json:"event_type"`
	IssueAddress   string `json:"issue_address"`
	Decimals       int    `json:"decimals"`
	TokenName      string `json:"token_name"`
	ID             string `json:"id"`
	Direction      int    `json:"direction"`
}

type EtherscanResult struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	From              string `json:"from"`
	ContractAddress   string `json:"contractAddress"`
	To                string `json:"to"`
	Value             string `json:"value"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Input             string `json:"input"`
	Confirmations     string `json:"confirmations"`
}

func Trc20ApiScan(token string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Trc20CallBack:", time.Now().UTC().Format("2006-01-02 15:04:05 MST"), err)
			log.Sugar.Error(err)
		}
	}()
	tokenWithChainPrefix := "trc20:" + token
	if !data.IsWalletLocked(tokenWithChainPrefix) {
		return
	}
	client := http_client.GetHttpClient()
	startTime := carbon.Now().AddHours(-24).TimestampWithMillisecond()
	endTime := carbon.Now().TimestampWithMillisecond()
	resp, err := client.R().SetQueryParams(map[string]string{
		"sort":            "-timestamp",
		"limit":           "50",
		"start":           "0",
		"direction":       "2",
		"db_version":      "1",
		"trc20Id":         "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"address":         token,
		"start_timestamp": stdutil.ToString(startTime),
		"end_timestamp":   stdutil.ToString(endTime),
	}).Get(UsdtTrc20ApiUri)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != http.StatusOK {
		panic(resp.StatusCode())
	}
	var trc20Resp UsdtTrc20Resp
	err = json.Cjson.Unmarshal(resp.Body(), &trc20Resp)
	if err != nil {
		panic(err)
	}
	if trc20Resp.PageSize <= 0 {
		return
	}
	for _, transfer := range trc20Resp.Data {
		if transfer.To != token || transfer.ContractRet != "SUCCESS" {
			continue
		}
		decimalQuant, err := decimal.NewFromString(transfer.Amount)
		if err != nil {
			panic(err)
		}
		decimalDivisor := decimal.NewFromFloat(1000000)
		amount := decimalQuant.Div(decimalDivisor).InexactFloat64()
		tradeId, err := data.GetTradeIdByWalletAddressAndAmount(tokenWithChainPrefix, amount)
		if err != nil {
			panic(err)
		}
		if tradeId == "" {
			continue
		}
		order, err := data.GetOrderInfoByTradeId(tradeId)
		if err != nil {
			panic(err)
		}
		// 区块的确认时间必须在订单创建时间之后
		createTime := order.CreatedAt.TimestampWithMillisecond()
		if transfer.BlockTimestamp < createTime {
			log.Sugar.Warnf("Orders cannot actually be matched: %s <-> %s", tradeId, transfer.Hash)
			continue
		}
		// 到这一步就完全算是支付成功了
		req := &request.OrderProcessingRequest{
			TokenWithChainPrefix: tokenWithChainPrefix,
			TradeId:              tradeId,
			Amount:               amount,
			BlockTransactionId:   transfer.Hash,
		}
		err = OrderProcessing(req)
		if err != nil {
			panic(err)
		}
		// 回调队列
		orderCallbackQueue, _ := handle.NewOrderCallbackQueue(order)
		mq.MClient.Enqueue(orderCallbackQueue, asynq.MaxRetry(5))
		// 发送机器人消息
		msgTpl := `
<b>📢📢有新的 Trc20 交易支付成功！</b>
<pre>交易号：%s</pre>
<pre>订单号：%s</pre>
<pre>请求支付金额：%f cny</pre>
<pre>实际支付金额：%f usdt</pre>
<pre>钱包地址：%s</pre>
<pre>订单创建时间：%s</pre>
<pre>支付成功时间：%s</pre>
<pre>交易哈希：%s</pre>
`
		msg := fmt.Sprintf(msgTpl,
			order.TradeId, order.OrderId, order.Amount, order.ActualAmount, tokenWithChainPrefix, order.CreatedAt.ToDateTimeString(), carbon.Now().ToDateTimeString(), transfer.Hash)
		telegram.SendToBot(msg)
	}
}

func EtherscanApiScan(chainName, token string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("EtherscanCallBack:", time.Now().UTC().Format("2006-01-02 15:04:05 MST"), err)
			log.Sugar.Error(err)
		}
	}()
	var chainId string
	switch chainName {
	case model.ChainNamePolygonPOS:
		chainId = "137"
	case model.ChainNameBSC:
		chainId = "56"
	case model.ChainNameAVAXC:
		chainId = "43114"
	case model.ChainNameETH:
		chainId = "1"
	case model.ChainNameArbitrum:
		chainId = "42161"
	default:
		return
	}
	var usdtContract string
	switch chainName {
	case model.ChainNamePolygonPOS:
		usdtContract = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
	case model.ChainNameBSC:
		usdtContract = "0x55d398326f99059fF775485246999027B3197955"
	case model.ChainNameAVAXC:
		usdtContract = "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7"
	case model.ChainNameETH:
		usdtContract = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	case model.ChainNameArbitrum:
		usdtContract = "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"
	default:
		return
	}
	var decimalDivisor decimal.Decimal
	switch chainName {
	case model.ChainNameBSC:
		decimalDivisor = decimal.NewFromFloat(1000000000000000000) // 18
	default:
		decimalDivisor = decimal.NewFromFloat(1000000) // 6
	}
	tokenWithChainPrefix := chainName + ":" + token
	if !data.IsWalletLocked(tokenWithChainPrefix) {
		return
	}
	client := http_client.GetHttpClient()
	apiKey := config.GetEtherscanApi()
	resp, err := client.R().SetQueryParams(map[string]string{
		"chainid": chainId,
		"module":  "account",
		"action":  "tokentx",
		"address": token,
		"page":    "1",
		"offset":  "10",
		"sort":    "desc",
		"apiKey":  apiKey,
	}).Get(EtherscanApiUri)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode() != http.StatusOK {
		panic(resp.StatusCode())
	}
	//println(resp.String())
	var etherscanResp EtherscanResp
	body := resp.Body()
	err = json.Cjson.Unmarshal(body, &etherscanResp)
	if err != nil {
		panic(err)
	}
	if etherscanResp.Status != "1" {
		panic(string(body))
	}
	for _, transfer := range etherscanResp.Data {
		confirmation, _ := strconv.Atoi(transfer.Confirmations)
		// EVM 地址不区分大小写
		isUSDT := strings.EqualFold(transfer.ContractAddress, usdtContract)
		isToThisAccount := strings.EqualFold(transfer.To, token)
		if !isUSDT || !isToThisAccount || confirmation < 5 {
			// fmt.Println("不符合条件的转账:", transfer)
			continue
		}
		decimalQuant, err := decimal.NewFromString(transfer.Value)
		if err != nil {
			panic(err)
		}
		amount := decimalQuant.Div(decimalDivisor).InexactFloat64()
		tradeId, err := data.GetTradeIdByWalletAddressAndAmount(tokenWithChainPrefix, amount)
		if err != nil {
			panic(err)
		}
		if tradeId == "" {
			continue
		}
		order, err := data.GetOrderInfoByTradeId(tradeId)
		if err != nil {
			panic(err)
		}
		// 区块的确认时间必须在订单创建时间之后
		createTime := order.CreatedAt.TimestampWithSecond()
		timestamp, err := strconv.ParseInt(transfer.TimeStamp, 10, 64)
		if err != nil {
			panic(err)
		}
		if timestamp < createTime {
			log.Sugar.Warnf("Orders cannot actually be matched: %s <-> %s", tradeId, transfer.Hash)
			continue
		}
		// 到这一步就完全算是支付成功了
		req := &request.OrderProcessingRequest{
			TokenWithChainPrefix: tokenWithChainPrefix,
			TradeId:              tradeId,
			Amount:               amount,
			BlockTransactionId:   transfer.Hash,
		}
		err = OrderProcessing(req)
		if err != nil {
			panic(err)
		}
		// 回调队列
		orderCallbackQueue, _ := handle.NewOrderCallbackQueue(order)
		_, _ = mq.MClient.Enqueue(orderCallbackQueue, asynq.MaxRetry(5))
		// 发送机器人消息
		msgTpl := `
<b>📢📢有新的交易支付成功！</b>
<pre>交易号：%s</pre>
<pre>订单号：%s</pre>
<pre>请求支付金额：%f cny</pre>
<pre>实际支付金额：%f usdt</pre>
<pre>钱包地址：%s</pre>
<pre>订单创建时间：%s</pre>
<pre>支付成功时间：%s</pre>
<pre>交易哈希：%s</pre>
`
		msg := fmt.Sprintf(msgTpl,
			order.TradeId, order.OrderId, order.Amount, order.ActualAmount, tokenWithChainPrefix, order.CreatedAt.ToDateTimeString(), carbon.Now().ToDateTimeString(), transfer.Hash)
		telegram.SendToBot(msg)
	}
}
