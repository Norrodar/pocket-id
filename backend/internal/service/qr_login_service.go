package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const QrLoginSessionDuration = 5 * time.Minute

type QrLoginService struct {
	db              *gorm.DB
	jwtService      *JwtService
	auditLogService *AuditLogService
}

func NewQrLoginService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService) *QrLoginService {
	return &QrLoginService{
		db:              db,
		jwtService:      jwtService,
		auditLogService: auditLogService,
	}
}

// CreateSession generates a new QR login session with a random token.
func (s *QrLoginService) CreateSession(ctx context.Context) (string, int, error) {
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", 0, err
	}

	session := &model.QrLoginSession{
		Token:        token,
		ExpiresAt:    datatype.DateTime(time.Now().Add(QrLoginSessionDuration)),
		IsAuthorized: false,
	}

	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return "", 0, err
	}

	return token, int(QrLoginSessionDuration.Seconds()), nil
}

// GetSessionStatus checks whether a QR login session has been authorized.
func (s *QrLoginService) GetSessionStatus(ctx context.Context, token string) (bool, error) {
	var session model.QrLoginSession
	err := s.db.
		WithContext(ctx).
		First(&session, "token = ?", token).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, &common.TokenInvalidOrExpiredError{}
		}
		return false, err
	}

	if time.Now().After(session.ExpiresAt.ToTime()) {
		return false, &common.TokenInvalidOrExpiredError{}
	}

	return session.IsAuthorized, nil
}

// ConfirmSession is called from the phone after passkey authentication to authorize the session.
func (s *QrLoginService) ConfirmSession(ctx context.Context, token string, userID string, ipAddress string, userAgent string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var session model.QrLoginSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&session, "token = ?", token).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &common.TokenInvalidOrExpiredError{}
		}
		return fmt.Errorf("error finding QR login session: %w", err)
	}

	if time.Now().After(session.ExpiresAt.ToTime()) {
		return &common.TokenInvalidOrExpiredError{}
	}

	if session.IsAuthorized {
		return &common.TokenInvalidOrExpiredError{}
	}

	session.UserID = &userID
	session.IsAuthorized = true

	if err := tx.WithContext(ctx).Save(&session).Error; err != nil {
		return fmt.Errorf("error saving QR login session: %w", err)
	}

	s.auditLogService.Create(ctx, model.AuditLogEventQrLoginSignIn, ipAddress, userAgent, userID, model.AuditLogData{}, tx)

	return tx.Commit().Error
}

// ExchangeSession is called from the TV after the session has been confirmed.
// It exchanges the session for a JWT access token and deletes the session.
func (s *QrLoginService) ExchangeSession(ctx context.Context, token string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var session model.QrLoginSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		First(&session, "token = ?", token).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", &common.TokenInvalidOrExpiredError{}
		}
		return model.User{}, "", err
	}

	if time.Now().After(session.ExpiresAt.ToTime()) {
		return model.User{}, "", &common.TokenInvalidOrExpiredError{}
	}

	if !session.IsAuthorized || session.UserID == nil {
		return model.User{}, "", &common.TokenInvalidOrExpiredError{}
	}

	accessToken, err := s.jwtService.GenerateAccessToken(session.User)
	if err != nil {
		return model.User{}, "", err
	}

	// Delete session (single-use)
	if err := tx.WithContext(ctx).Delete(&session).Error; err != nil {
		return model.User{}, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return model.User{}, "", err
	}

	return session.User, accessToken, nil
}
