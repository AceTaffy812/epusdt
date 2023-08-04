package telegram

import (
	"fmt"
	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/mdb"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
	tb "gopkg.in/telebot.v3"
	"strings"
)

const (
	ReplayAddWallet = "请输入钱包地址以及链名, 中间用空格分开, 例如: \nTNEns8t9jbWENbStkQdVQtHMGpbsYsQjZK trc20\n0xb91dd8225Db88dE4E3CD7B7eC538677A2c1Be8Cb polygon"
)

func OnTextMessageHandle(c tb.Context) error {
	if c.Message().ReplyTo.Text == ReplayAddWallet {
		defer bots.Delete(c.Message().ReplyTo)
		info := strings.Split(c.Message().Text, " ")
		walletAddress := info[0]
		channel := info[1]
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
	for _, wallet := range wallets {
		status := "已启用✅"
		if wallet.Status == mdb.TokenStatusDisable {
			status = "已禁用🚫"
		}
		var temp []tb.InlineButton
		btnInfo := tb.InlineButton{
			Unique: wallet.Token,
			Text:   fmt.Sprintf("%s[%s]", wallet.Token, status),
			Data:   strutil.MustString(wallet.ID),
		}
		bots.Handle(&btnInfo, WalletInfo)
		btnList = append(btnList, append(temp, btnInfo))
	}
	addBtn := tb.InlineButton{Text: "添加钱包地址", Unique: "AddWallet"}
	bots.Handle(&addBtn, func(c tb.Context) error {
		return c.Send(ReplayAddWallet, &tb.ReplyMarkup{
			ForceReply: true,
		})
	})
	btnList = append(btnList, []tb.InlineButton{addBtn})
	return c.EditOrSend("请点击钱包继续操作", &tb.ReplyMarkup{
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
