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
	token := orderInfo.TokenWithChainPrefix
	if strings.Count(token, ":") == 1 {
		parts := strings.Split(token, ":")
		channel = parts[0]
		token = parts[1]
	}
	switch channel {
	case model.ChainNamePolygonPOS:
		channel = "Polygon PoS Chain (POL)"
	case model.ChainNameAVAXC:
		channel = "Avalanche (C-Chain)"
	case model.ChainNameETH:
		channel = "Ethereum - ERC20"
	case model.ChainNameBSC:
		channel = "BNB Smart Chain - BEP20"
	case model.ChainNameTRC20:
		channel = "TRON - TRC20"
	case model.ChainNameAptos:
		channel = "Aptos"
	case model.ChainNameArbitrum:
		channel = "Arbitrum One"
	}
	resp := &response.CheckoutCounterResponse{
		TradeId:        orderInfo.TradeId,
		ActualAmount:   orderInfo.ActualAmount,
		Channel:        channel,
		Token:          token,
		ExpirationTime: orderInfo.CreatedAt.AddMinutes(config.GetOrderExpirationTime()).TimestampWithMillisecond(),
		RedirectUrl:    orderInfo.RedirectUrl,
	}
	return resp, nil
}
