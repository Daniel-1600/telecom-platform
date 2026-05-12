package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nutcas3/telecom-platform/apps/api-server/internal/models"
	"github.com/nutcas3/telecom-platform/apps/api-server/internal/services"
)

// AuCHandler handles Authentication Center (AuC) HTTP requests.
type AuCHandler struct {
	subscriberService *services.SubscriberService
}

// NewAuCHandler creates a new AuCHandler.
func NewAuCHandler(subscriberService *services.SubscriberService) *AuCHandler {
	return &AuCHandler{subscriberService: subscriberService}
}

// GenerateAuthVector generates a 3GPP EPS authentication vector for a subscriber.
//
// @Summary      Generate authentication vector
// @Description  Runs the Milenage algorithm and returns (RAND, XRES, CK, IK, AUTN).
//
//	Called by the MME/AMF during an Attach or TAU procedure.
//
// @Tags         auc
// @Produce      json
// @Param        imsi  path      string  true  "Subscriber IMSI (15 digits)"
// @Success      200   {object}  services.AuthVector
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/v1/auc/{imsi}/auth-vector [post]
func (h *AuCHandler) GenerateAuthVector(c *gin.Context) {
	imsi := models.IMSI(c.Param("imsi"))
	if imsi == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "IMSI is required",
			Code:    ErrCodeMissingRequired,
			Details: "imsi path parameter must not be empty",
		})
		return
	}

	av, err := h.subscriberService.GenerateAuthVector(c.Request.Context(), imsi)
	if err != nil {
		handleError(c, err, "Failed to generate authentication vector", ErrCodeInternalError)
		return
	}

	c.JSON(http.StatusOK, av)
}

// RegisterAuCRoutes registers AuC routes on the given router group.
// Call this from your router setup alongside other handler registrations.
//
//	v1 := r.Group("/api/v1")
//	handlers.RegisterAuCRoutes(v1, handlers.NewAuCHandler(subscriberService))
func RegisterAuCRoutes(rg *gin.RouterGroup, h *AuCHandler) {
	auc := rg.Group("/auc")
	{
		auc.POST("/:imsi/auth-vector", h.GenerateAuthVector)
	}
}
