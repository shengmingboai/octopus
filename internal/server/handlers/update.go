package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shengmingboai/octopus/internal/conf"
	"github.com/shengmingboai/octopus/internal/server/resp"
	"github.com/shengmingboai/octopus/internal/update"
)

func latest(c *gin.Context) {
	latestInfo, err := update.GetLatestInfo()
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, *latestInfo)
}

func getNowVersion(c *gin.Context) {
	resp.Success(c, conf.Version)
}

func updateFunc(c *gin.Context) {
	err := update.UpdateCore()
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, "update success")
}
