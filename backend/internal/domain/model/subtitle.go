package model

import "github.com/google/uuid"

type Subtitle struct {
	Base
	VideoID  uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_subtitles_video_language_format,priority:1" json:"video_id"`
	Language string    `gorm:"size:16;not null;index;uniqueIndex:idx_subtitles_video_language_format,priority:2" json:"language"`
	Format   string    `gorm:"size:16;not null;default:'srt';uniqueIndex:idx_subtitles_video_language_format,priority:3" json:"format"`
	Content  string    `gorm:"type:text" json:"content"`
	FilePath string    `gorm:"type:text" json:"file_path"`

	Video Video `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
