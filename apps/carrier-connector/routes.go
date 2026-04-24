package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/es2"
	handler "github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/handler"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/repository"
)

// setupRoutes registers all HTTP routes, wiring the ES2+ client and profile repo.
func setupRoutes(router *gin.Engine, client *es2.ES2Client, repo repository.ProfileRepository) {
	api := router.Group("/api/v1")
	api.GET("/health", healthHandler)

	esim := api.Group("/esim")
	{
		esim.POST("/profiles", handler.OrderProfileHandlerWithRepo(client, repo))
		esim.GET("/profiles", handler.ListProfilesHandlerWithRepo(repo))
		esim.GET("/profiles/:profileId", handler.GetProfileHandlerWithRepo(client, repo))
		esim.DELETE("/profiles/:profileId", handler.DeleteProfileHandlerWithRepo(client, repo))
	}

	carrier := api.Group("/carrier")
	{
		carrier.GET("/info", handler.GetCarrierInfoHandler(client))
		carrier.GET("/connectivity", handler.CheckConnectivityHandler(client))
	}
}

// healthHandler returns a simple liveness response.
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "carrier-connector",
		"timestamp": time.Now().UTC(),
	})
}
