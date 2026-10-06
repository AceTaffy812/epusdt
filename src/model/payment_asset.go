package model

import (
	"strings"
)

const (
	AssetUSDT = "usdt"
	AssetUSDC = "usdc"
)

// PaymentAsset is the built-in, immutable definition of a payment asset.
// Wallets remain associated with Chain only; an asset is selected by channel.
type PaymentAsset struct {
	Chain         string
	Symbol        string
	ChainID       string
	Contract      string
	Decimals      int
	Confirmations int // 只对 EVM 生效
	AmountStep    float64
}

var paymentAssets = []PaymentAsset{
	// USDT
	{Chain: ChainNameETH, Symbol: AssetUSDT, ChainID: "1", Contract: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNamePlasma, Symbol: AssetUSDT, ChainID: "9745", Contract: "0xB8CE59FC3717ada4C02eaDF9682A9e934F625ebb", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNameAVAXC, Symbol: AssetUSDT, ChainID: "43114", Contract: "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNameBSC, Symbol: AssetUSDT, ChainID: "56", Contract: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNamePolygonPOS, Symbol: AssetUSDT, ChainID: "137", Contract: "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNameAptos, Symbol: AssetUSDT, Contract: "0x357b0b74bc833e95a115ad22604854d6b0fca151cecd94111770e5d6ffc9dc2b", Decimals: 6, Confirmations: 0, AmountStep: 0.01},
	{Chain: ChainNameTRC20, Symbol: AssetUSDT, Contract: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", Decimals: 6, Confirmations: 0, AmountStep: 0.01},
	// USDC
	{Chain: ChainNameETH, Symbol: AssetUSDC, ChainID: "1", Contract: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNamePlasma, Symbol: AssetUSDC, ChainID: "9745", Contract: "0x2d661C89D812261039AF9764eceaAee884f5F67F", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNameAVAXC, Symbol: AssetUSDC, ChainID: "43114", Contract: "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNameBSC, Symbol: AssetUSDC, ChainID: "56", Contract: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Decimals: 18, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNamePolygonPOS, Symbol: AssetUSDC, ChainID: "137", Contract: "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", Decimals: 6, Confirmations: 5, AmountStep: 0.01},
	{Chain: ChainNameAptos, Symbol: AssetUSDC, Contract: "0xbae207659db88bea0cbead6da0ed00aac12edcdda169e591cd41c94180b46f3b", Decimals: 6, Confirmations: 0, AmountStep: 0.01},
}

// ParsePaymentChannel keeps every legacy channel as USDT. A composite channel
// (for example polygon_usdc) selects a registered asset without changing API fields.
func ParsePaymentChannel(channel string) (PaymentAsset, bool) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = ChainNamePolygonPOS
	}
	chain, symbol := channel, AssetUSDT
	if parts := strings.Split(channel, "_"); len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		chain, symbol = parts[0], parts[1]
	} else if strings.Contains(channel, "_") {
		return PaymentAsset{}, false
	}
	for _, asset := range paymentAssets {
		if asset.Chain == chain && asset.Symbol == symbol {
			return asset, true
		}
	}
	return PaymentAsset{}, false
}

func GetPaymentAsset(chain, symbol string) (PaymentAsset, bool) {
	return ParsePaymentChannel(strings.ToLower(chain) + "_" + strings.ToLower(symbol))
}

func EnabledPaymentAssets(chain string, enabled func(string, string) bool) []PaymentAsset {
	assets := make([]PaymentAsset, 0)
	for _, asset := range paymentAssets {
		if asset.Chain == chain && enabled(asset.Chain, asset.Symbol) {
			assets = append(assets, asset)
		}
	}
	return assets
}

// PaymentAssets returns a copy of every built-in payment asset definition.
func PaymentAssets() []PaymentAsset {
	assets := make([]PaymentAsset, len(paymentAssets))
	copy(assets, paymentAssets)
	return assets
}

// PaymentChainDisplayName returns the human-readable name shown at checkout.
func PaymentChainDisplayName(chain string) string {
	switch chain {
	case ChainNamePolygonPOS:
		return "Polygon PoS Chain (POL)"
	case ChainNameAVAXC:
		return "Avalanche (C-Chain)"
	case ChainNameETH:
		return "Ethereum - ERC20"
	case ChainNameBSC:
		return "BNB Smart Chain - BEP20"
	case ChainNameTRC20:
		return "TRON - TRC20"
	case ChainNameAptos:
		return "Aptos"
	case ChainNamePlasma:
		return "Plasma"
	default:
		return chain
	}
}

// PaymentAssetDisplayName returns the human-readable symbol used by payment pages.
func PaymentAssetDisplayName(symbol string) string {
	switch strings.ToLower(symbol) {
	case AssetUSDT:
		return "USDT"
	case AssetUSDC:
		return "USDC"
	default:
		return strings.ToUpper(symbol)
	}
}
