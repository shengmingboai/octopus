package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shengmingboai/octopus/internal/model"
	"github.com/shengmingboai/octopus/internal/op"
	"github.com/shengmingboai/octopus/internal/server/resp"
)

type statsDailyResponse struct {
	MaxRequestCount int64              `json:"max_request_count"` // 最近 54 周每日请求量的最大值。
	Items           []model.StatsDaily `json:"items"`             // 每日原始统计数据。
}

func getStatsDaily(c *gin.Context) {
	now := time.Now()
	since := now.AddDate(0, 0, -(int(now.Weekday()) + 53*7)).Format("20060102")
	statsDaily, err := op.StatsGetDaily(c.Request.Context(), since)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	maxRequestCount := int64(0)
	for _, daily := range statsDaily {
		maxRequestCount = max(maxRequestCount, daily.RequestSuccess+daily.RequestFailed)
	}

	resp.Success(c, statsDailyResponse{
		MaxRequestCount: maxRequestCount,
		Items:           statsDaily,
	})
}

func getStatsHourly(c *gin.Context) {
	resp.Success(c, op.StatsHourlyGet())
}

func getStatsTotal(c *gin.Context) {
	resp.Success(c, op.StatsTotalGet())
}

func getStatsAPIKey(c *gin.Context) {
	resp.Success(c, op.StatsAPIKeyList())
}
