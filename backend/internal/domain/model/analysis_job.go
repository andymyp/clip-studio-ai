package model

import "github.com/google/uuid"

type AnalysisJob struct {
	Base
	VideoID  uuid.UUID `gorm:"type:uuid;not null;index;index:idx_analysis_jobs_video_status,priority:1" json:"video_id"`
	Status   JobStatus `gorm:"size:32;not null;default:'pending';index;index:idx_analysis_jobs_video_status,priority:2" json:"status"`
	Progress float64   `gorm:"not null;default:0;check:chk_analysis_jobs_progress_range,progress >= 0 AND progress <= 100" json:"progress"`
	Message  string    `gorm:"type:text" json:"message"`

	Video Video `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
