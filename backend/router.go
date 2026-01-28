package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

// setupRouter はGinルーターを設定
func setupRouter() *gin.Engine {
	// 環境に応じてGinモードを設定
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// グローバルミドルウェア
	r.Use(gin.Recovery())
	r.Use(RequestLoggerMiddleware())
	r.Use(NgrokHeadersMiddleware())

	// ========== Bot Group ==========
	r.POST("/webhook", LineSignatureMiddleware(), handleWebhook)

	// ========== API Group ==========
	api := r.Group("/api")
	{
		// LIFF endpoints - LIFFトークン認証が必要
		liff := api.Group("/liff")
		liff.Use(LIFFAuthMiddleware())
		{
			liff.POST("/register", handleRegisterUser)
			liff.POST("/message", handleLIFFMessage)
			liff.GET("/me", handleGetMyInfo)
			liff.GET("/events", handleGetEvents)
			liff.POST("/events", handleCreateEvent)
			liff.GET("/approvals", handleGetApprovals)
			liff.POST("/approvals", handleApprovePayments)
			liff.GET("/circle/members", handleGetCircleMembers) // レガシー互換

			// サークル管理（新API）
			liff.GET("/circles", handleGetMyCircles)
			liff.POST("/circles", handleCreateCircle)
			liff.POST("/circles/join", handleJoinCircle)
			liff.GET("/circles/search", handleSearchCircles)
			liff.GET("/circles/:id/members", handleGetCircleMembersByID)
			liff.POST("/circles/:id/leave", handleLeaveCircle)
			liff.POST("/circles/:id/remove", handleRemoveFromCircle)
			liff.POST("/circles/:id/primary", handleSetPrimaryCircle)
		}

		// Admin endpoints - APIキー認証が必要
		admin := api.Group("/admin")
		admin.Use(AdminAuthMiddleware())
		{
			admin.GET("/users", handleGetUsers)
			admin.GET("/messages", handleAllMessages)
			admin.POST("/send", handleSend)
			admin.POST("/test/send-reminders", handleTestReminder)

			// リッチメニュー管理
			admin.GET("/richmenu", handleRichMenuList)
			admin.POST("/richmenu", handleRichMenuCreate)
			admin.POST("/richmenu/:id/image", handleRichMenuImageUpload)
			admin.POST("/richmenu/:id/default", handleRichMenuSetDefault)
			admin.DELETE("/richmenu/:id", handleRichMenuDelete)
		}
	}

	// ========== 静的ファイル ==========
	// NoRouteはSPAルーティングをサポート - 不明なパスにはindex.htmlを返す
	r.NoRoute(handleStaticFiles)

	return r
}
