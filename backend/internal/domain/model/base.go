package model

import (
	"time"

	"github.com/google/uuid"
)

// Base contains fields shared by every persisted domain model.
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// EnsureID generates IDs in the application so persistence does not depend on
// a PostgreSQL UUID extension.
func (base *Base) EnsureID() {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
}
