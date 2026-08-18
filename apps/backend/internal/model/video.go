package model

import "github.com/google/uuid"

type Video struct {
	Base
	UserID          uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_videos_user_platform_url" json:"user_id"`
	ExternalID      string    `gorm:"size:160;index" json:"external_id"`
	YouTubeUsername string    `gorm:"size:100" json:"youtube_username"`
	License         string    `gorm:"size:100" json:"license"`
	Reusable        bool      `gorm:"not null;default:false" json:"reusable"`
	Platform        string    `gorm:"size:50;not null;uniqueIndex:idx_videos_user_platform_url" json:"platform"`
	URL             string    `gorm:"type:text;not null;uniqueIndex:idx_videos_user_platform_url" json:"url"`
	Title           string    `gorm:"type:text;not null" json:"title"`
	Description     string    `gorm:"type:text" json:"description"`
	Thumbnail       string    `gorm:"type:text" json:"thumbnail"`
	Duration        float64   `gorm:"not null;default:0;check:chk_videos_duration_nonnegative,duration >= 0" json:"duration"`
	Views           int64     `gorm:"not null;default:0;check:chk_videos_views_nonnegative,views >= 0" json:"views"`
	Likes           int64     `gorm:"not null;default:0;check:chk_videos_likes_nonnegative,likes >= 0" json:"likes"`
	Comments        int64     `gorm:"not null;default:0;check:chk_videos_comments_nonnegative,comments >= 0" json:"comments"`
	Transcript      string    `gorm:"type:text" json:"transcript"`

	User  User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Clips []Clip `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
