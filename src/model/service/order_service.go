package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/dao"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/request"
	"github.com/assimon/luuu/model/response"
	"github.com/assimon/luuu/mq"
	"github.com/assimon/luuu/mq/handle"
	"github.com/assimon/luuu/util/constant"
	"github.com/assimon/luuu/util/math"
	"github.com/golang-module/carbon/v2"
	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
)

const (
	CnyMinimumPaymentAmount  = 0.01 // cny最低支付金额
	UsdtMinimumPaymentAmount = 0.01 // usdt最低支付金额
	UsdtAmountPerIncrement   = 0.01 // usdt每次递增金额
	IncrementalMaximumNumber = 100  // 最大递增次数
)

var gCreateTransactionLock sync.Mutex

const checkoutOrderCacheKeyPrefix = "epusdt:checkout-order:"

// CheckoutOrderAsset is a selectable currency under a payment network.
type CheckoutOrderAsset struct {
	Symbol  string
	Channel string
}

// CheckoutOrderChain is a selectable payment network and its available assets.
type CheckoutOrderChain struct {
	Name   string
	Assets []CheckoutOrderAsset
}

// CheckoutOrderPage contains the data required to render the payment-method selector.
type CheckoutOrderPage struct {
	CheckoutId string
	OrderId    string
	Amount     float64
	ReturnUrl  string
	Chains     []CheckoutOrderChain
}

func paymentURL(path string) string {
	base := strings.TrimRight(config.GetAppUri(), "/")
	if base == "" {
		return path
	}
	return base + path
}

type pendingCheckoutOrder struct {
	OrderId      string  `json:"order_id"`
	Amount       float64 `json:"amount"`
	NotifyUrl    string  `json:"notify_url"`
	ExchangeRate string  `json:"exchange_rate"`
	RedirectUrl  string  `json:"redirect_url"`
}

// CreateTransaction 创建订单
func CreateTransaction(req *request.CreateTransactionRequest) (*response.CreateTransactionResponse, error) {
	return createTransaction(req, "")
}

// createTransaction creates the order using tradeID when it is supplied. The
// checkout flow uses its checkout ID as the final trade ID so that both API
// responses refer to the same order. An empty tradeID keeps the legacy ID
// generation behavior for the direct transaction API.
func createTransaction(req *request.CreateTransactionRequest, tradeID string) (*response.CreateTransactionResponse, error) {
	gCreateTransactionLock.Lock()
	defer gCreateTransactionLock.Unlock()
	_, decimalUsdt, err := calculateOrderUsdtAmount(req)
	if err != nil {
		return nil, err
	}
	// 已经存在了的交易
	if err = ensureOrderDoesNotExist(req.OrderId); err != nil {
		return nil, err
	}
	asset, ok := model.ParsePaymentChannel(req.Channel)
	if !ok || !config.IsPaymentAssetEnabled(asset.Chain, asset.Symbol) {
		return nil, constant.NotAvailableWalletAddress
	}
	// Wallet records always contain the base chain, never a composite channel.
	walletAddress, err := data.GetAvailableWallet(asset.Chain)
	if err != nil {
		return nil, err
	}
	if len(walletAddress) <= 0 {
		return nil, constant.NotAvailableWalletAddress
	}

	amount := math.MustParsePrecFloat64(decimalUsdt.InexactFloat64(), 2)
	availableToken, availableAmount, err := CalculateAvailableWalletAndAmount(amount, asset, walletAddress)
	if err != nil {
		return nil, err
	}
	if availableToken == "" {
		return nil, constant.NotAvailableAmountErr
	}
	tx := dao.Mdb.Begin()
	if tradeID == "" {
		tradeID = GenerateCode()
	}
	order := &mdb.Orders{
		TradeId:       tradeID,
		OrderId:       req.OrderId,
		Amount:        req.Amount,
		ActualAmount:  availableAmount,
		WalletAddress: asset.Chain + ":" + availableToken,
		Asset:         asset.Symbol,
		Status:        mdb.StatusWaitPay,
		NotifyUrl:     req.NotifyUrl,
		RedirectUrl:   req.RedirectUrl,
	}
	err = data.CreateOrderWithTransaction(tx, order)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	// 锁定支付池
	err = data.LockTransaction(asset.Chain, asset.Symbol, availableToken, order.TradeId, availableAmount, config.GetOrderExpirationTimeDuration())
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	// 超时过期消息队列
	orderExpirationQueue, _ := handle.NewOrderExpirationQueue(order.TradeId)
	mq.MClient.Enqueue(orderExpirationQueue, asynq.ProcessIn(config.GetOrderExpirationTimeDuration()))
	ExpirationTime := carbon.Now().AddMinutes(config.GetOrderExpirationTime()).Timestamp()
	resp := &response.CreateTransactionResponse{
		TradeId:        order.TradeId,
		OrderId:        order.OrderId,
		Amount:         order.Amount,
		ActualAmount:   order.ActualAmount,
		Token:          order.WalletAddress,
		ExpirationTime: ExpirationTime,
		PaymentUrl:     paymentURL(fmt.Sprintf("/pay/checkout-counter/%s", order.TradeId)),
	}
	return resp, nil
}

// CreateCheckoutOrder stores a signed order request until the payer chooses a network and asset.
// The caller-supplied channel is deliberately ignored.
func CreateCheckoutOrder(req *request.CreateTransactionRequest) (*response.CreateOrderResponse, error) {
	if _, _, err := calculateOrderUsdtAmount(req); err != nil {
		return nil, err
	}
	if err := ensureOrderDoesNotExist(req.OrderId); err != nil {
		return nil, err
	}
	pending := pendingCheckoutOrder{
		OrderId: req.OrderId, Amount: req.Amount, NotifyUrl: req.NotifyUrl,
		ExchangeRate: req.ExchangeRate, RedirectUrl: req.RedirectUrl,
	}
	payload, err := json.Marshal(pending)
	if err != nil {
		return nil, err
	}
	// Keep the checkout ID in the same format and length as the orders.trade_id
	// column because it becomes the final order's trade ID after selection.
	checkoutId := GenerateCode()
	if err = dao.Rdb.Set(context.Background(), checkoutOrderCacheKeyPrefix+checkoutId, payload, config.GetOrderExpirationTimeDuration()).Err(); err != nil {
		return nil, err
	}
	return &response.CreateOrderResponse{
		Fiat: "CNY", TradeId: checkoutId, OrderId: req.OrderId,
		Amount: strconv.FormatFloat(req.Amount, 'f', 2, 64), Status: strconv.Itoa(mdb.StatusWaitPay),
		ExpirationTime: int64(config.GetOrderExpirationTimeDuration() / time.Second),
		PaymentUrl:     paymentURL(fmt.Sprintf("/pay/checkout-order/%s", checkoutId)),
	}, nil
}

// GetCheckoutOrderPage returns only enabled assets on networks that have an enabled wallet.
func GetCheckoutOrderPage(checkoutId string) (*CheckoutOrderPage, error) {
	pending, err := getPendingCheckoutOrder(checkoutId)
	if err != nil {
		return nil, err
	}
	chains := make([]CheckoutOrderChain, 0)
	assetsByChain := make(map[string][]CheckoutOrderAsset)
	chainOrder := make([]string, 0)
	for _, asset := range model.PaymentAssets() {
		if !config.IsPaymentAssetEnabled(asset.Chain, asset.Symbol) {
			continue
		}
		if _, known := assetsByChain[asset.Chain]; !known {
			wallets, walletErr := data.GetAvailableWallet(asset.Chain)
			if walletErr != nil {
				return nil, walletErr
			}
			if len(wallets) == 0 {
				assetsByChain[asset.Chain] = nil
				continue
			}
			assetsByChain[asset.Chain] = make([]CheckoutOrderAsset, 0)
			chainOrder = append(chainOrder, asset.Chain)
		}
		if assetsByChain[asset.Chain] == nil {
			continue
		}
		// The checkout always submits the explicit composite channel. This keeps
		// the selected asset unambiguous, including USDT (for example plasma_usdt).
		channel := asset.Chain + "_" + asset.Symbol
		assetsByChain[asset.Chain] = append(assetsByChain[asset.Chain], CheckoutOrderAsset{
			Symbol: model.PaymentAssetDisplayName(asset.Symbol), Channel: channel,
		})
	}
	chainOrder = orderCheckoutChains(chainOrder, assetsByChain, config.GetPaymentChainOrder())
	for _, chain := range chainOrder {
		assets := assetsByChain[chain]
		if len(assets) == 0 {
			continue
		}
		chains = append(chains, CheckoutOrderChain{
			Name: model.PaymentChainDisplayName(chain), Assets: assets,
		})
	}
	return &CheckoutOrderPage{
		CheckoutId: checkoutId,
		OrderId:    pending.OrderId,
		Amount:     pending.Amount,
		ReturnUrl:  pending.RedirectUrl,
		Chains:     chains,
	}, nil
}

// orderCheckoutChains applies the configured network order while retaining
// networks that are not explicitly listed for backwards compatibility.
func orderCheckoutChains(current []string, assetsByChain map[string][]CheckoutOrderAsset, preferred []string) []string {
	if len(preferred) == 0 {
		return current
	}

	ordered := make([]string, 0, len(current))
	seen := make(map[string]struct{}, len(current))
	appendIfAvailable := func(chain string) {
		if len(assetsByChain[chain]) == 0 {
			return
		}
		if _, ok := seen[chain]; ok {
			return
		}
		seen[chain] = struct{}{}
		ordered = append(ordered, chain)
	}
	for _, chain := range preferred {
		appendIfAvailable(chain)
	}
	for _, chain := range current {
		appendIfAvailable(chain)
	}
	return ordered
}

// CreateTransactionFromCheckout creates the final order with the payer-selected channel.
func CreateTransactionFromCheckout(checkoutId, channel string) (*response.CreateTransactionResponse, error) {
	pending, err := getPendingCheckoutOrder(checkoutId)
	if err != nil {
		return nil, err
	}
	transaction := &request.CreateTransactionRequest{
		OrderId: pending.OrderId, Amount: pending.Amount, NotifyUrl: pending.NotifyUrl,
		ExchangeRate: pending.ExchangeRate, RedirectUrl: pending.RedirectUrl, Channel: channel,
	}
	resp, err := createTransaction(transaction, checkoutId)
	if err != nil {
		return nil, err
	}
	if err = dao.Rdb.Del(context.Background(), checkoutOrderCacheKeyPrefix+checkoutId).Err(); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCheckoutOrderReturnURL returns the merchant callback URL kept with a
// pending checkout. It is used when the payer-facing checkout cannot proceed.
func GetCheckoutOrderReturnURL(checkoutId string) string {
	pending, err := getPendingCheckoutOrder(checkoutId)
	if err != nil {
		return ""
	}
	return pending.RedirectUrl
}

func getPendingCheckoutOrder(checkoutId string) (*pendingCheckoutOrder, error) {
	payload, err := dao.Rdb.Get(context.Background(), checkoutOrderCacheKeyPrefix+checkoutId).Bytes()
	if err != nil {
		return nil, errors.New("收银台已失效，请重新发起支付")
	}
	pending := new(pendingCheckoutOrder)
	if err = json.Unmarshal(payload, pending); err != nil {
		return nil, errors.New("收银台数据无效")
	}
	return pending, nil
}

func calculateOrderUsdtAmount(req *request.CreateTransactionRequest) (float64, decimal.Decimal, error) {
	payAmount := math.MustParsePrecFloat64(req.Amount, 2)
	decimalRate, err := decimal.NewFromString(req.ExchangeRate)
	if err != nil || decimalRate.LessThanOrEqual(decimal.Zero) {
		decimalRate = decimal.NewFromFloat(config.GetUsdtRate())
	}
	decimalPayAmount := decimal.NewFromFloat(payAmount)
	decimalUsdt := decimalPayAmount.Div(decimalRate)
	if decimalPayAmount.Cmp(decimal.NewFromFloat(CnyMinimumPaymentAmount)) == -1 ||
		decimalUsdt.Cmp(decimal.NewFromFloat(UsdtMinimumPaymentAmount)) == -1 {
		return 0, decimal.Zero, constant.PayAmountErr
	}
	return payAmount, decimalUsdt, nil
}

func ensureOrderDoesNotExist(orderId string) error {
	exist, err := data.GetOrderInfoByOrderId(orderId)
	if err != nil {
		return err
	}
	if exist.ID > 0 {
		return constant.OrderAlreadyExists
	}
	return nil
}

// OrderProcessing 成功处理订单
func OrderProcessing(req *request.OrderProcessingRequest) error {
	tx := dao.Mdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	exist, err := data.GetOrderByBlockIdWithTransaction(tx, req.BlockTransactionId)
	if err != nil {
		tx.Rollback()
		return err
	}
	if exist.ID > 0 {
		tx.Rollback()
		return constant.OrderBlockAlreadyProcess
	}
	// 标记订单成功
	err = data.OrderSuccessWithTransaction(tx, req)
	if err != nil {
		tx.Rollback()
		return err
	}
	// 解锁交易
	err = data.UnLockTransaction(req.Chain, req.Asset, req.Address, req.Amount)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// CalculateAvailableWalletAndAmount 计算可用钱包地址和金额
func CalculateAvailableWalletAndAmount(amount float64, asset model.PaymentAsset, walletAddress []mdb.WalletAddress) (string, float64, error) {
	availableToken := ""
	availableAmount := amount
	calculateAvailableWalletFunc := func(amount float64) (string, error) {
		availableWallet := ""
		for _, address := range walletAddress {
			result, err := data.GetTradeIdByWalletAddressAndAmount(asset.Chain, asset.Symbol, address.Token, amount)
			if err != nil {
				return "", err
			}
			if result == "" {
				availableWallet = address.Token
				break
			}
		}
		return availableWallet, nil
	}
	for i := 0; i < IncrementalMaximumNumber; i++ {
		token, err := calculateAvailableWalletFunc(availableAmount)
		if err != nil {
			return "", 0, err
		}
		// 拿不到可用钱包就累加金额
		if token == "" {
			decimalOldAmount := decimal.NewFromFloat(availableAmount)
			decimalIncr := decimal.NewFromFloat(asset.AmountStep)
			availableAmount = decimalOldAmount.Add(decimalIncr).InexactFloat64()
			continue
		}
		availableToken = token
		break
	}
	return availableToken, availableAmount, nil
}

// GenerateCode 订单号生成
func GenerateCode() string {
	date := time.Now().Format("20060102")
	r := rand.Intn(1000)
	code := fmt.Sprintf("%s%d%03d", date, time.Now().UnixNano()/1e6, r)
	return code
}

// GetOrderInfoByTradeId 通过交易号获取订单
func GetOrderInfoByTradeId(tradeId string) (*mdb.Orders, error) {
	order, err := data.GetOrderInfoByTradeId(tradeId)
	if err != nil {
		return nil, err
	}
	if order.ID <= 0 {
		return nil, constant.OrderNotExists
	}
	return order, nil
}
