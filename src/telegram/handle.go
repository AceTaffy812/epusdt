package telegram

import (
	"fmt"
	"strings"

	"github.com/assimon/luuu/model"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
	tb "gopkg.in/telebot.v3"
)

const (
	ReplayAddWallet = "请输入钱包地址, 目前仅支持 trc20 eth polygon bsc avax-c aptos plasma 链。"
)

func OnTextMessageHandle(c tb.Context) error {
	if c.Message().ReplyTo.Text == ReplayAddWallet {
		defer bots.Delete(c.Message().ReplyTo)
		walletAddress := c.Message().Text
		var channel = ""
		if strings.HasPrefix(walletAddress, "T") {
			channel = model.ChainNameTRC20
		} else if strings.HasPrefix(walletAddress, "0x") {
			return c.Send("EVM 系列钱包地址请在地址前加上所属链和英文冒号，以区分不同的链，例如 eth: polygon: bsc: avax-c: plasma: aptos:")
		} else if strings.HasPrefix(walletAddress, "polygon:0x") {
			channel = model.ChainNamePolygonPOS
			walletAddress = strings.TrimPrefix(walletAddress, "polygon:")
		} else if strings.HasPrefix(walletAddress, "bsc:0x") {
			channel = model.ChainNameBSC
			walletAddress = strings.TrimPrefix(walletAddress, "bsc:")
		} else if strings.HasPrefix(walletAddress, "avax-c:0x") {
			channel = model.ChainNameAVAXC
			walletAddress = strings.TrimPrefix(walletAddress, "avax-c:")
		} else if strings.HasPrefix(walletAddress, "eth:0x") {
			channel = model.ChainNameETH
			walletAddress = strings.TrimPrefix(walletAddress, "eth:")
		} else if strings.HasPrefix(walletAddress, "plasma:0x") {
			channel = model.ChainNamePlasma
			walletAddress = strings.TrimPrefix(walletAddress, "plasma:")
		} else if strings.HasPrefix(walletAddress, "aptos:0x") {
			channel = model.ChainNameAptos
			walletAddress = strings.TrimPrefix(walletAddress, "aptos:")
		} else {
			return c.Send("不支持该钱包地址！")
		}
		_, err := data.AddWalletAddress(walletAddress, channel)
		if err != nil {
			return c.Send(err.Error())
		}
		c.Send(fmt.Sprintf("钱包[%s]添加成功！", c.Message().Text))
		return WalletList(c)
	}
	return nil
}

func WalletList(c tb.Context) error {
	wallets, err := data.GetAllWalletAddress()
	if err != nil {
		return err
	}

	var btnList [][]tb.InlineButton
	var fullList strings.Builder
	fullList.WriteString("请点击钱包继续操作\n\n")
	fullList.WriteString("完整钱包地址列表：\n")

	for i, wallet := range wallets {
		status := "已启用✅"
		if wallet.Status == mdb.TokenStatusDisable {
			status = "已禁用🚫"
		}

		// 按钮显示内容（截断）
		tokenShow := wallet.Token
		if len(wallet.Token) > 50 {
			tokenShow = wallet.Token[:50]
		}

		// --- 按钮 ---
		var temp []tb.InlineButton
		btnInfo := tb.InlineButton{
			Unique: strutil.Md5(wallet.Token),
			Text:   fmt.Sprintf("[%s] %s [%s]", wallet.Channel, tokenShow, status),
			Data:   strutil.MustString(wallet.ID),
		}
		bots.Handle(&btnInfo, WalletInfo)
		btnList = append(btnList, append(temp, btnInfo))

		// --- 追加完整地址到消息内容 ---
		fullList.WriteString(
			fmt.Sprintf("%d. [%s] %s\n", i+1, wallet.Channel, wallet.Token),
		)
	}

	// 添加钱包按钮
	addBtn := tb.InlineButton{Text: "添加钱包地址", Unique: "AddWallet"}
	bots.Handle(&addBtn, func(c tb.Context) error {
		return c.Send(ReplayAddWallet, &tb.ReplyMarkup{
			ForceReply: true,
		})
	})
	btnList = append(btnList, []tb.InlineButton{addBtn})

	return c.EditOrSend(fullList.String(), &tb.ReplyMarkup{
		InlineKeyboard: btnList,
	})
}

func WalletInfo(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	tokenInfo, err := data.GetWalletAddressById(id)
	if err != nil {
		return c.Send(err.Error())
	}
	enableBtn := tb.InlineButton{
		Text:   "启用",
		Unique: "enableBtn",
		Data:   c.Data(),
	}
	disableBtn := tb.InlineButton{
		Text:   "禁用",
		Unique: "disableBtn",
		Data:   c.Data(),
	}
	delBtn := tb.InlineButton{
		Text:   "删除",
		Unique: "delBtn",
		Data:   c.Data(),
	}
	backBtn := tb.InlineButton{
		Text:   "返回",
		Unique: "WalletList",
	}
	bots.Handle(&enableBtn, EnableWallet)
	bots.Handle(&disableBtn, DisableWallet)
	bots.Handle(&delBtn, DelWallet)
	bots.Handle(&backBtn, WalletList)
	return c.EditOrReply(tokenInfo.Token, &tb.ReplyMarkup{InlineKeyboard: [][]tb.InlineButton{
		{
			enableBtn,
			disableBtn,
			delBtn,
		},
		{
			backBtn,
		},
	}})
}

func EnableWallet(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	if id <= 0 {
		return c.Send("请求不合法！")
	}
	err := data.ChangeWalletAddressStatus(id, mdb.TokenStatusEnable)
	if err != nil {
		return c.Send(err.Error())
	}
	return WalletList(c)
}

func DisableWallet(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	if id <= 0 {
		return c.Send("请求不合法！")
	}
	err := data.ChangeWalletAddressStatus(id, mdb.TokenStatusDisable)
	if err != nil {
		return c.Send(err.Error())
	}
	return WalletList(c)
}

func DelWallet(c tb.Context) error {
	id := mathutil.MustUint(c.Data())
	if id <= 0 {
		return c.Send("请求不合法！")
	}
	err := data.DeleteWalletAddressById(id)
	if err != nil {
		return c.Send(err.Error())
	}
	return WalletList(c)
}
