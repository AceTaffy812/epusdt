package comm

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model/response"
	"github.com/assimon/luuu/model/service"
	"github.com/labstack/echo/v4"
)

var paymentErrorTemplate = template.Must(template.New("payment-error").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        * { box-sizing: border-box; }
        body { margin: 0; min-height: 100vh; display: grid; place-items: center; padding: 24px; background: #f4f6f8; color: #1f2937; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
        .error { width: min(100%, 480px); padding: 32px 24px; background: #fff; border-radius: 16px; box-shadow: 0 10px 30px rgba(15, 23, 42, .08); text-align: center; }
        h1 { margin: 0 0 16px; color: #b91c1c; font-size: 22px; }
        p { margin: 0; line-height: 1.7; overflow-wrap: anywhere; }
        a { display: inline-block; margin-top: 24px; padding: 10px 18px; border-radius: 8px; background: #2563eb; color: #fff; text-decoration: none; }
    </style>
</head>
<body>
    <main class="error" role="alert">
        <h1>{{.Title}}</h1>
        <p>{{.Message}}</p>
        {{if .ReturnURL}}
        <a href="{{.ReturnURL}}" id="return-link">返回商户页面</a>
        {{end}}
    </main>
</body>
</html>`))

// renderPaymentError returns a readable error page instead of an unstyled blank
// response when a checkout or payment page cannot be built.
func renderPaymentError(ctx echo.Context, status int, title string, err error, returnURL string) error {
	message := "页面加载失败，请稍后重试。"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = err.Error()
	}
	data := struct {
		Title     string
		Message   string
		ReturnURL string
	}{Title: title, Message: message, ReturnURL: returnURL}
	var body bytes.Buffer
	if executeErr := paymentErrorTemplate.Execute(&body, data); executeErr != nil {
		return ctx.String(status, message)
	}
	return ctx.HTML(status, body.String())
}

// CheckoutCounter 收银台
func (c *BaseCommController) CheckoutCounter(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	resp, err := service.GetCheckoutCounterByTradeId(tradeId)
	if err != nil {
		return renderPaymentError(ctx, http.StatusNotFound, "支付页面加载失败", err, "")
	}
	tmpl, err := template.ParseFiles(fmt.Sprintf(".%s/%s", config.StaticPath, "pay.html"))
	if err != nil {
		return renderPaymentError(ctx, http.StatusInternalServerError, "支付页面加载失败", err, "")
	}
	var body bytes.Buffer
	if err = tmpl.Execute(&body, resp); err != nil {
		return renderPaymentError(ctx, http.StatusInternalServerError, "支付页面加载失败", err, "")
	}
	return ctx.HTML(http.StatusOK, body.String())
}

// CheckStatus 支付状态检测
func (c *BaseCommController) CheckStatus(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	order, err := service.GetOrderInfoByTradeId(tradeId)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	resp := response.CheckStatusResponse{
		TradeId: order.TradeId,
		Status:  order.Status,
	}
	return c.SucJson(ctx, resp)
}

// CheckoutOrder renders the network and asset selection page for a pending order.
func (c *BaseCommController) CheckoutOrder(ctx echo.Context) (err error) {
	checkoutId := ctx.Param("checkout_id")
	resp, err := service.GetCheckoutOrderPage(checkoutId)
	if err != nil {
		return renderPaymentError(ctx, http.StatusNotFound, "收银台加载失败", err, service.GetCheckoutOrderReturnURL(checkoutId))
	}
	tmpl, err := template.ParseFiles(fmt.Sprintf(".%s/%s", config.StaticPath, "checkout-order.html"))
	if err != nil {
		return renderPaymentError(ctx, http.StatusInternalServerError, "收银台加载失败", err, resp.ReturnUrl)
	}
	var body bytes.Buffer
	if err = tmpl.Execute(&body, resp); err != nil {
		return renderPaymentError(ctx, http.StatusInternalServerError, "收银台加载失败", err, resp.ReturnUrl)
	}
	return ctx.HTML(http.StatusOK, body.String())
}

// ConfirmCheckoutOrder creates the final transaction from the payer-selected asset.
func (c *BaseCommController) ConfirmCheckoutOrder(ctx echo.Context) (err error) {
	checkoutId := ctx.Param("checkout_id")
	channel := ctx.FormValue("channel")
	resp, err := service.CreateTransactionFromCheckout(checkoutId, channel)
	if err != nil {
		return renderPaymentError(ctx, http.StatusBadRequest, "创建支付订单失败", err, service.GetCheckoutOrderReturnURL(checkoutId))
	}
	return ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("../checkout-counter/%s", resp.TradeId))
}
