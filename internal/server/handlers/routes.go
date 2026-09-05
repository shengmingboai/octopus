package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/shengmingboai/octopus/internal/relay"
	"github.com/shengmingboai/octopus/internal/server/middleware"
)

// RegisterRoutes 显式注册接口及认证边界，每个 Engine 都拥有完整、独立的路由表。
func RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/user/login", middleware.RequireJSON(), login)

	admin := r.Group("/api/v1", middleware.Auth())
	user := admin.Group("/user", middleware.RequireJSON())
	user.POST("/change-password", changePassword)
	user.POST("/change-username", changeUsername)
	user.GET("/status", status)

	apikey := admin.Group("/apikey", middleware.RequireJSON())
	apikey.POST("/create", createAPIKey)
	apikey.GET("/list", listAPIKey)
	apikey.POST("/update", updateAPIKey)
	apikey.DELETE("/delete/:id", deleteAPIKey)

	channel := admin.Group("/channel", middleware.RequireJSON())
	channel.GET("/detail/:id", getChannelDetail)
	channel.GET("/stats", listChannelStats)
	channel.GET("/grants", listChannelGrant)
	channel.POST("/create", createChannel)
	channel.POST("/update", updateChannel)
	channel.POST("/enable", enableChannel)
	channel.DELETE("/delete/:id", deleteChannel)
	channel.POST("/fetch-model", fetchModel)
	channel.POST("/sync/:id", syncChannelModels)
	channel.POST("/sync-all", syncAllChannels)
	channel.GET("/last-sync-time", getLastSyncTime)

	group := admin.Group("/group", middleware.RequireJSON())
	group.GET("/list", getGroupList)
	group.GET("/get/:id", getGroup)
	group.GET("/events", streamGroupEvents)
	group.POST("/create", createGroup)
	group.POST("/update/:id", updateGroup)
	group.DELETE("/delete/:id", deleteGroup)

	model := admin.Group("/model", middleware.RequireJSON())
	model.GET("/list", listLLM)
	model.POST("/create", createLLM)
	model.POST("/update", updateLLM)
	model.POST("/delete", deleteLLM)
	model.POST("/update-price", updateLLMPrice)
	model.POST("/rebuild-price", rebuildLLMPrice)
	model.GET("/last-update-time", getLastUpdateTime)

	stats := admin.Group("/stats")
	stats.GET("/daily", getStatsDaily)
	stats.GET("/hourly", getStatsHourly)
	stats.GET("/total", getStatsTotal)
	stats.GET("/apikey", getStatsAPIKey)

	logs := admin.Group("/log")
	logs.GET("/overview/stream", streamOverview)
	logs.GET("/:id/request-body", getRequestBody)
	logs.GET("/:id/response-body", getResponseBody)
	logs.POST("/:request_id/:round/stop", interruptRound)
	logs.DELETE("/clear", clearLog)

	setting := admin.Group("/setting")
	setting.GET("/list", getSettingList)
	setting.POST("/set", middleware.RequireJSON(), setSetting)
	setting.GET("/export", exportDB)
	setting.POST("/import", importDB)

	update := admin.Group("/update")
	update.GET("", latest)
	update.GET("/now-version", getNowVersion)
	update.POST("", updateFunc)

	dashboard := r.Group("/api/v1/apikey", middleware.APIKeyAuth())
	dashboard.GET("/stats", getStatsAPIKeyById)
	dashboard.GET("/login", loginAPIKey)

	api := r.Group("/v1", middleware.APIKeyAuth())
	api.GET("/models", getModelList)
	api.POST("/chat/completions", relay.Forward(llm.APIFormatOpenAIChatCompletion))
	api.POST("/responses", relay.Forward(llm.APIFormatOpenAIResponse))
	api.POST("/messages", relay.Forward(llm.APIFormatAnthropicMessage))
}
