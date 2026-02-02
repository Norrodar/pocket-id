package model

import (
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type QrLoginSession struct {
	Base
	Token        string
	ExpiresAt    datatype.DateTime
	IsAuthorized bool

	UserID *string
	User   User
}
