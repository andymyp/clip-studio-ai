package model

import "github.com/google/uuid"

type RenderJob struct {
	Base
	ClipID            uuid.UUID `gorm:"type:uuid;not null;index;index:idx_render_jobs_clip_status,priority:1" json:"clip_id"`
	Status            JobStatus `gorm:"size:32;not null;default:'pending';index;index:idx_render_jobs_clip_status,priority:2" json:"status"`
	Progress          float64   `gorm:"not null;default:0" json:"progress"`
	Message           string    `gorm:"type:text" json:"message"`
	OutputPath        string    `gorm:"type:text" json:"output_path"`
	MediaURL          string    `gorm:"type:text" json:"media_url"`
	SubtitlePath      string    `gorm:"type:text" json:"subtitle_path"`
	WatermarkText     string    `gorm:"type:text" json:"watermark_text"`
	SourceURL         string    `gorm:"type:text" json:"source_url"`
	SourceUsername    string    `gorm:"size:100" json:"source_username"`
	Title             string    `gorm:"type:text" json:"title"`
	Description       string    `gorm:"type:text" json:"description"`
	Hashtags          string    `gorm:"type:text" json:"hashtags"`
	Error             string    `gorm:"type:text" json:"error"`
	Hook              string    `gorm:"type:text" json:"hook"`
	RemovedSeconds    float64   `gorm:"not null;default:0" json:"removed_seconds"`
	PatternInterrupts int       `gorm:"not null;default:0" json:"pattern_interrupts"`
	PlaybackSpeed     float64   `gorm:"not null;default:1" json:"playback_speed"`
	PlatformProfile   string    `gorm:"size:32;not null;default:'smart';index" json:"platform_profile"`
	RightsConfirmed   bool      `gorm:"not null;default:false" json:"rights_confirmed"`
	QualityPassed     bool      `gorm:"not null;default:false" json:"quality_passed"`
	OutputWidth       int       `gorm:"not null;default:0" json:"output_width"`
	OutputHeight      int       `gorm:"not null;default:0" json:"output_height"`
	OutputDuration    float64   `gorm:"not null;default:0" json:"output_duration"`
	AudioVideoDrift   float64   `gorm:"not null;default:0" json:"audio_video_drift"`
	AudioLoudnessLUFS float64   `gorm:"not null;default:0" json:"audio_loudness_lufs"`

	Clip        Clip              `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Performance []ClipPerformance `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
