package model

import "github.com/google/uuid"

type Watermark struct {
	Base
	UserID   uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name     string    `gorm:"size:160;not null" json:"name"`
	FilePath string    `gorm:"type:text;not null" json:"file_path"`
	Position string    `gorm:"size:32;not null;default:'bottom-right'" json:"position"`
	Opacity  float64   `gorm:"not null;default:1;check:chk_watermarks_opacity_range,opacity >= 0 AND opacity <= 1" json:"opacity"`
	Scale    float64   `gorm:"not null;default:1;check:chk_watermarks_scale_positive,scale > 0" json:"scale"`

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
