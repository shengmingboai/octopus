package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/shengmingboai/octopus/internal/conf"
	"github.com/shengmingboai/octopus/internal/server/handlers"
	"github.com/shengmingboai/octopus/internal/server/middleware"
	"github.com/shengmingboai/octopus/internal/server/resp"
	"github.com/shengmingboai/octopus/static"
)

var httpSrv *http.Server
var cancelRequests context.CancelFunc

func newRouter() *gin.Engine {
	if conf.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		resp.Error(c, http.StatusInternalServerError, resp.ErrInternalServer)
		c.Abort()
	}))

	if conf.IsDebug() {
		r.Use(middleware.Logger())
	}
	r.Use(middleware.Cors())
	r.Use(middleware.StaticEmbed("/", static.StaticFS))

	handlers.RegisterRoutes(r)
	return r
}

func Start() error {
	addr := net.JoinHostPort(conf.AppConfig.Server.Host, strconv.Itoa(conf.AppConfig.Server.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelRequests = cancel
	httpSrv = &http.Server{
		Addr:        addr,
		Handler:     newRouter(),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	srv := httpSrv
	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("http server listen and serve error: %v", err)
		}
	}()
	return nil
}

func Close() error {
	if httpSrv == nil {
		return nil
	}
	// 先取消长连接和上游请求，等待处理器完成统计后再关闭存储。
	cancelRequests()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		return errors.Join(err, httpSrv.Close())
	}
	return nil
}
