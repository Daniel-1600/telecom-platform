package mvno

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Router sets up the MVNO routing and middleware
type Router struct {
	handler *Handler
	logger  *logrus.Logger
}

// NewRouter creates a new MVNO router
func NewRouter(db *gorm.DB, logger *logrus.Logger) *Router {
	// Create repository
	repo := NewGormRepository(db, logger)
	
	// Create service
	service := NewOnboardingService(logger)
	
	// Create handler
	handler := NewHandler(service, repo, logger)
	
	return &Router{
		handler: handler,
		logger:  logger,
	}
}

// SetupRoutes configures all MVNO routes with middleware
func (r *Router) SetupRoutes(router *gin.Engine) {
	// API versioning group
	v1 := router.Group("/api/v1")
	
	// Add middleware for logging and recovery
	v1.Use(gin.Logger())
	v1.Use(gin.Recovery())
	
	// Register MVNO routes
	r.handler.RegisterRoutes(v1)
	
	r.logger.Info("MVNO routes configured")
}

// SetupMiddleware configures middleware for MVNO endpoints
func (r *Router) SetupMiddleware(router *gin.Engine) {
	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// Rate limiting middleware (basic implementation)
	router.Use(func(c *gin.Context) {
		// In production, implement proper rate limiting
		c.Next()
	})
	
	// Request ID middleware
	router.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = "mvno-" + c.Request.URL.Path
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})
}

// HealthCheck provides a simple health check endpoint
func (r *Router) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"service": "mvno-onboarding",
		"version": "1.0.0",
	})
}

// RegisterHealthCheck registers the health check endpoint
func (r *Router) RegisterHealthCheck(router *gin.Engine) {
	router.GET("/health/mvno", r.HealthCheck)
	r.logger.Info("MVNO health check endpoint registered")
}
