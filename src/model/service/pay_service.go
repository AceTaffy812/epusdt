package service

import (
	"errors"
	"strings"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/assimon/luuu/model/response"
)

// GetCheckoutCounterByTradeId 获取收银台详情，通过订单
func GetCheckoutCounterByTradeId(tradeId string) (*response.CheckoutCounterResponse, error) {
	orderInfo, err := data.GetOrderInfoByTradeId(tradeId)
	if err != nil {
		return nil, err
	}
	if orderInfo.ID <= 0 || orderInfo.Status != mdb.StatusWaitPay {
		return nil, errors.New("不存在待支付订单或已过期！")
	}
	channel := ""
	token := orderInfo.WalletAddress
	if strings.Count(token, ":") == 1 {
		parts := strings.Split(token, ":")
		channel = parts[0]
		token = parts[1]
	}
	baseChannel := channel
	channel = model.PaymentChainDisplayName(baseChannel)
	assetSymbol := orderInfo.Asset
	if assetSymbol == "" {
		assetSymbol = model.AssetUSDT
	}
	asset, _ := model.GetPaymentAsset(baseChannel, assetSymbol)
	resp := &response.CheckoutCounterResponse{
		TradeId:        orderInfo.TradeId,
		ActualAmount:   orderInfo.ActualAmount,
		Chain:          channel,
		Asset:          model.PaymentAssetDisplayName(assetSymbol),
		AssetIcon:      paymentAssetIcon(assetSymbol),
		Contract:       asset.Contract,
		Token:          token,
		ExpirationTime: orderInfo.CreatedAt.AddMinutes(config.GetOrderExpirationTime()).TimestampWithMillisecond(),
		RedirectUrl:    orderInfo.RedirectUrl,
	}
	return resp, nil
}

// paymentAssetIcon returns the static icon matching the asset shown on the payment page.
func paymentAssetIcon(symbol string) string {
	var icon string
	switch strings.ToLower(strings.TrimSpace(symbol)) {
	case model.AssetUSDT:
		icon = "Usdt--Streamline-Cryptocurrency.svg"
	case model.AssetUSDC:
		icon = "Usdc--Streamline-Cryptocurrency.svg"
	default:
		return ""
	}
	return paymentURL(strings.TrimRight(config.StaticPath, "/") + "/" + icon)
}
