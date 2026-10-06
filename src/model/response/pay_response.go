package response

type CheckoutCounterResponse struct {
	TradeId        string  `json:"trade_id"`        //  epusdt订单号
	ActualAmount   float64 `json:"actual_amount"`   //  订单实际需要支付的金额，保留4位小数
	Chain          string  `json:"chain"`           //  收款钱包网络
	Asset          string  `json:"asset"`           // 支付币种
	AssetIcon      string  `json:"asset_icon"`      // 支付币种图标地址
	Contract       string  `json:"contract"`        // 合约或链上资产 Metadata 地址
	Token          string  `json:"token"`           //  收款钱包地址
	ExpirationTime int64   `json:"expiration_time"` // 过期时间 时间戳
	RedirectUrl    string  `json:"redirect_url"`
}

type CheckStatusResponse struct {
	TradeId string `json:"trade_id"` //  epusdt订单号
	Status  int    `json:"status"`
}
