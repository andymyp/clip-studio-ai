package model

import "github.com/google/uuid"

type Clip struct {
	Base
	VideoID   uuid.UUID  `gorm:"type:uuid;not null;index;index:idx_clips_video_status,priority:1" json:"video_id"`
	StartTime float64    `gorm:"not null;check:chk_clips_start_nonnegative,start_time >= 0" json:"start_time"`
	EndTime   float64    `gorm:"not null;check:chk_clips_time_range,end_time > start_time" json:"end_time"`
	Score     float64    `gorm:"not null;default:0;index;check:chk_clips_score_range,score >= 0 AND score <= 100" json:"score"`
	Reason    string     `gorm:"type:text" json:"reason"`
	Status    ClipStatus `gorm:"size:32;not null;default:'candidate';index;index:idx_clips_video_status,priority:2" json:"status"`

	Video      Video       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	RenderJobs []RenderJob `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
