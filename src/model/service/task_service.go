package service

import (
	"container/list"
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
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
)

const EtherscanApiUri = "https://api.etherscan.io/v2/api"

const unmatchedOrderLogCacheSize = 1000

var unmatchedOrderLogs = struct {
	sync.Mutex
	entries map[string]*list.Element
	order   *list.List
}{
	entries: make(map[string]*list.Element),
	order:   list.New(),
}

func warnOrderCannotActuallyBeMatchedOnce(chain, tradeID, transactionID string) {
	logKey := fmt.Sprintf("%s:%s:%s", chain, tradeID, transactionID)

	unmatchedOrderLogs.Lock()
	if _, exists := unmatchedOrderLogs.entries[logKey]; exists {
		unmatchedOrderLogs.Unlock()
		return
	}
	unmatchedOrderLogs.entries[logKey] = unmatchedOrderLogs.order.PushBack(logKey)
	if unmatchedOrderLogs.order.Len() > unmatchedOrderLogCacheSize {
		oldest := unmatchedOrderLogs.order.Front()
		delete(unmatchedOrderLogs.entries, oldest.Value.(string))
		unmatchedOrderLogs.order.Remove(oldest)
	}
	unmatchedOrderLogs.Unlock()

	log.Sugar.Warnf("Orders cannot actually be matched: %s <-> %s", tradeID, transactionID)
}

type EtherscanResp struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Data    []EtherscanResult `json:"result"`
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

func EtherscanApiScan(asset model.PaymentAsset, token string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("EtherscanCallBack:", time.Now().UTC().Format("2006-01-02 15:04:05 MST"), err)
			log.Sugar.Error(err)
		}
	}()
	if asset.ChainID == "" || asset.Contract == "" {
		return
	}
	decimalDivisor := decimal.New(1, int32(asset.Decimals))
	if !data.IsWalletLocked(asset.Chain, asset.Symbol, token) {
		return
	}
	client := http_client.GetHttpClient()
	apiKey := config.GetEtherscanApi()
	resp, err := client.R().SetQueryParams(map[string]string{
		"chainid": asset.ChainID,
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
	if etherscanResp.Status != "1" && len(etherscanResp.Data) > 0 {
		panic(string(body))
	}
	for _, transfer := range etherscanResp.Data {
		confirmation, _ := strconv.Atoi(transfer.Confirmations)
		// EVM 地址不区分大小写
		isTargetAsset := strings.EqualFold(transfer.ContractAddress, asset.Contract)
		isToThisAccount := strings.EqualFold(transfer.To, token)
		if !isTargetAsset || !isToThisAccount || confirmation < asset.Confirmations {
			// fmt.Println("不符合条件的转账:", transfer)
			continue
		}
		decimalQuant, err := decimal.NewFromString(transfer.Value)
		if err != nil {
			panic(err)
		}
		amount := decimalQuant.Div(decimalDivisor).InexactFloat64()
		tradeId, err := data.GetTradeIdByWalletAddressAndAmount(asset.Chain, asset.Symbol, token, amount)
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
			warnOrderCannotActuallyBeMatchedOnce(asset.Chain, tradeId, transfer.Hash)
			continue
		}
		// 到这一步就完全算是支付成功了
		req := &request.OrderProcessingRequest{
			Address:            token,
			TradeId:            tradeId,
			Amount:             amount,
			BlockTransactionId: transfer.Hash,
			Chain:              asset.Chain,
			Asset:              asset.Symbol,
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
<pre>收款交易类型：%s</pre>
<pre>交易号：%s</pre>
<pre>订单号：%s</pre>
<pre>请求支付金额：%f cny</pre>
<pre>实际支付金额：%f %s</pre>
<pre>钱包地址：%s</pre>
<pre>订单创建时间：%s</pre>
<pre>支付成功时间：%s</pre>
<pre>交易哈希：%s</pre>
`
		msg := fmt.Sprintf(msgTpl,
			asset.Chain+"_"+asset.Symbol, order.TradeId, order.OrderId, order.Amount, order.ActualAmount, strings.ToUpper(asset.Symbol), order.WalletAddress, order.CreatedAt.ToDateTimeString(), carbon.Now().ToDateTimeString(), transfer.Hash)
		telegram.SendToBot(msg)
	}
}
