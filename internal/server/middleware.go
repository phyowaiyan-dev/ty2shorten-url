package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/middleware"
)

func registerCoreMiddleware(router *gin.Engine, logger *slog.Logger) {
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.LimitRequestBody())
}
