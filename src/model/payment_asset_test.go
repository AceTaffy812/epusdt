package model

import "testing"

func TestParsePaymentChannelKeepsLegacyUSDTDefaults(t *testing.T) {
	tests := []struct {
		channel string
		chain   string
	}{
		{"", ChainNamePolygonPOS},
		{ChainNamePolygonPOS, ChainNamePolygonPOS},
		{ChainNameTRC20, ChainNameTRC20},
		{ChainNameAptos, ChainNameAptos},
	}
	for _, test := range tests {
		asset, ok := ParsePaymentChannel(test.channel)
		if !ok || asset.Chain != test.chain || asset.Symbol != AssetUSDT {
			t.Fatalf("channel %q resolved to %#v, %v", test.channel, asset, ok)
		}
	}
}

func TestParsePaymentChannelUSDCAndUnsupportedAsset(t *testing.T) {
	asset, ok := ParsePaymentChannel("polygon_usdc")
	if !ok || asset.Chain != ChainNamePolygonPOS || asset.Symbol != AssetUSDC || asset.Decimals != 6 {
		t.Fatalf("polygon_usdc resolved to %#v, %v", asset, ok)
	}
	if _, ok := ParsePaymentChannel("trc20_usdc"); ok {
		t.Fatal("trc20_usdc must not be registered")
	}
	if _, ok := ParsePaymentChannel("polygon_usdc_extra"); ok {
		t.Fatal("malformed composite channel must not be accepted")
	}
}
