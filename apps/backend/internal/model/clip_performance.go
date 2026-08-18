package model

import "github.com/google/uuid"

type ClipPerformance struct {
	Base
	RenderJobID   uuid.UUID `gorm:"type:uuid;not null;index" json:"render_job_id"`
	Platform      string    `gorm:"size:32;not null;index" json:"platform"`
	Views         int64     `gorm:"not null;default:0" json:"views"`
	Likes         int64     `gorm:"not null;default:0" json:"likes"`
	Comments      int64     `gorm:"not null;default:0" json:"comments"`
	Shares        int64     `gorm:"not null;default:0" json:"shares"`
	AverageWatch  float64   `gorm:"not null;default:0" json:"average_watch_seconds"`
	CompletionPct float64   `gorm:"not null;default:0" json:"completion_percentage"`
	EngagedViews  int64     `gorm:"not null;default:0" json:"engaged_views"`
	SwipedAwayPct float64   `gorm:"not null;default:0" json:"swiped_away_percentage"`
	Replays       int64     `gorm:"not null;default:0" json:"replays"`
	DropoffSecond float64   `gorm:"not null;default:0" json:"dropoff_second"`
	ViralScore    float64   `gorm:"not null;default:0;index" json:"viral_score"`

	RenderJob RenderJob `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
