package model

import "github.com/google/uuid"

type RenderJob struct {
	Base
	ClipID     uuid.UUID `gorm:"type:uuid;not null;index;index:idx_render_jobs_clip_status,priority:1" json:"clip_id"`
	Status     JobStatus `gorm:"size:32;not null;default:'pending';index;index:idx_render_jobs_clip_status,priority:2" json:"status"`
	OutputPath string    `gorm:"type:text" json:"output_path"`

	Clip Clip `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
