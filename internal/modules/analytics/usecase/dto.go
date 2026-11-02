package usecase

import (
	"github.com/google/uuid"
)

type RecordViewInput struct {
	PostID     uuid.UUID
	SessionID  string
	DurationMS int
}
