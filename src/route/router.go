package route

import (
	"net/http"

	"github.com/assimon/luuu/controller/comm"
	"github.com/assimon/luuu/middleware"
	"github.com/labstack/echo/v4"
)

// RegisterRoute 路由注册
func RegisterRoute(e *echo.Echo) {
	e.Any("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello epusdt, https://github.com/assimon/epusdt")
	})
	// ==== 支付相关=====
	payRoute := e.Group("/pay")
	// 收银台
	payRoute.GET("/checkout-counter/:trade_id", comm.Ctrl.CheckoutCounter)
	// 支付方式选择收银台
	payRoute.GET("/checkout-order/:checkout_id", comm.Ctrl.CheckoutOrder)
	payRoute.POST("/checkout-order/:checkout_id", comm.Ctrl.ConfirmCheckoutOrder)
	// 状态检测
	payRoute.GET("/check-status/:trade_id", comm.Ctrl.CheckStatus)

	apiV1Route := e.Group("/api/v1")
	// ====订单相关====
	orderRoute := apiV1Route.Group("/order", middleware.CheckApiSign())
	// 创建订单
	orderRoute.POST("/create-transaction", comm.Ctrl.CreateTransaction)
	orderRoute.POST("/create-transaction/", comm.Ctrl.CreateTransaction)
	// 创建由付款方选择支付方式的订单；通配路由兼容会附加 API 路径的插件。
	orderRoute.POST("/create-order", comm.Ctrl.CreateOrder)
	orderRoute.POST("/create-order/*", comm.Ctrl.CreateOrder)
}
