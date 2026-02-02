package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

func NewQrLoginController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, qrLoginService *service.QrLoginService, appConfigService *service.AppConfigService) {
	qc := &QrLoginController{qrLoginService: qrLoginService, appConfigService: appConfigService}

	group.POST("/qr-login/init", rateLimitMiddleware.Add(rate.Every(10*time.Second), 5), qc.initSessionHandler)
	group.GET("/qr-login/status/:token", qc.getSessionStatusHandler)
	group.POST("/qr-login/confirm/:token", authMiddleware.WithAdminNotRequired().Add(), qc.confirmSessionHandler)
	group.POST("/qr-login/exchange/:token", rateLimitMiddleware.Add(rate.Every(10*time.Second), 5), qc.exchangeSessionHandler)
}

type QrLoginController struct {
	qrLoginService   *service.QrLoginService
	appConfigService *service.AppConfigService
}

// initSessionHandler creates a new QR login session
func (qc *QrLoginController) initSessionHandler(c *gin.Context) {
	token, expiresIn, err := qc.qrLoginService.CreateSession(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"expiresIn": expiresIn,
	})
}

// getSessionStatusHandler checks if a QR login session has been authorized
func (qc *QrLoginController) getSessionStatusHandler(c *gin.Context) {
	token := c.Param("token")

	authorized, err := qc.qrLoginService.GetSessionStatus(c.Request.Context(), token)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authorized": authorized,
	})
}

// confirmSessionHandler is called from the phone to authorize the QR login session
func (qc *QrLoginController) confirmSessionHandler(c *gin.Context) {
	token := c.Param("token")
	userID := c.GetString("userID")

	err := qc.qrLoginService.ConfirmSession(c.Request.Context(), token, userID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// exchangeSessionHandler exchanges an authorized QR login session for a JWT
func (qc *QrLoginController) exchangeSessionHandler(c *gin.Context) {
	token := c.Param("token")

	user, accessToken, err := qc.qrLoginService.ExchangeSession(c.Request.Context(), token)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(qc.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, accessToken)

	c.JSON(http.StatusOK, userDto)
}
