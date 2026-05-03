package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/es2"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/handlers"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/mq"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/repository"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/services"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/webhook"
)

// setupRoutes registers all HTTP routes, wiring the ES2+ client, profile repo, webhook client, and message queue.
func setupRoutes(router *gin.Engine, client *es2.ES2Client, profileRepo repository.ProfileRepository, webhookClient *webhook.WebhookClient, messageQueue *mq.MessageQueue, repo repository.Repository, logger *logrus.Logger) {
	api := router.Group("/api/v1")
	api.GET("/health", healthHandler)
	api.GET("/health/ready", readinessHandler(profileRepo))
	api.GET("/health/live", livenessHandler)
	api.GET("/metrics", gin.WrapH(promhttp.Handler()))

	esim := api.Group("/esim")
	{
		esim.POST("/profiles", handlers.OrderProfileHandlerWithRepo(client, profileRepo, webhookClient, messageQueue))
		esim.GET("/profiles", handlers.ListProfilesHandler(profileRepo))
		esim.GET("/profiles/:profileId", handlers.GetProfileHandler(profileRepo))
		esim.DELETE("/profiles/:profileId", handlers.DeleteProfileHandler(profileRepo))
	}

	carrier := api.Group("/carrier")
	{
		carrier.GET("/info", handlers.GetCarrierInfoHandler(client))
		carrier.GET("/connectivity", handlers.CheckConnectivityHandler(client))
	}

	// MVNO routes
	mvno := api.Group("/mvno")
	{
		// Create MVNO handlers
		onboardingService := services.NewOnboardingService(logger)
		mvnoHandler := handlers.NewMVNOHandler(onboardingService, repo, logger)
		managementHandler := handlers.NewManagementHandler(repo, logger)

		// MVNO onboarding and management routes
		mvno.POST("/onboarding", mvnoHandler.StartOnboarding)
		mvno.GET("", mvnoHandler.ListMVNOs)
		mvno.GET("/:id", mvnoHandler.GetMVNO)
		mvno.PUT("/:id/status", managementHandler.UpdateMVNOStatus)
		mvno.GET("/stats", managementHandler.GetMVNOStats)
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

// livenessHandler returns a simple liveness check (always healthy if service is running).
func livenessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   "carrier-connector",
		"timestamp": time.Now().UTC(),
	})
}

// readinessHandler checks if the service is ready to accept requests (database connectivity).
func readinessHandler(repo repository.ProfileRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check database connectivity
		if err := repo.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not ready",
				"service":   "carrier-connector",
				"timestamp": time.Now().UTC(),
				"error":     "database connection failed",
				"details":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"service":   "carrier-connector",
			"timestamp": time.Now().UTC(),
			"checks": gin.H{
				"database": "ok",
			},
		})
	}
}
