package model

import "time"

import "github.com/google/uuid"

type RecommendedVideo struct {
	Base
	ExternalID          string             `gorm:"size:32;not null;uniqueIndex" json:"external_id"`
	Platform            string             `gorm:"size:20;not null;index" json:"platform"`
	Category            string             `gorm:"size:80;not null;index:idx_recommendations_lookup" json:"category"`
	Language            string             `gorm:"size:16;not null;index:idx_recommendations_lookup" json:"language"`
	CategoryID          string             `gorm:"size:32" json:"category_id"`
	Title               string             `gorm:"type:text;not null" json:"title"`
	Description         string             `gorm:"type:text" json:"-"`
	ChannelID           string             `gorm:"size:64;index" json:"channel_id"`
	ChannelTitle        string             `gorm:"type:text" json:"channel_title"`
	YouTubeUsername     string             `gorm:"size:100" json:"youtube_username"`
	URL                 string             `gorm:"type:text;not null" json:"url"`
	EmbedURL            string             `gorm:"type:text" json:"embed_url"`
	Thumbnail           string             `gorm:"type:text" json:"thumbnail"`
	Duration            float64            `gorm:"not null;default:0" json:"duration"`
	Views               int64              `gorm:"not null;default:0" json:"views"`
	Likes               int64              `gorm:"not null;default:0" json:"likes"`
	Comments            int64              `gorm:"not null;default:0" json:"comments"`
	Subscribers         int64              `gorm:"not null;default:0" json:"subscribers"`
	PreviousSubscribers int64              `gorm:"not null;default:0" json:"-"`
	PublishedAt         time.Time          `gorm:"not null;index" json:"published_at"`
	CollectedAt         time.Time          `gorm:"not null;index" json:"collected_at"`
	ViralScore          float64            `gorm:"not null;default:0;index:idx_recommendations_lookup,sort:desc" json:"viral_score"`
	License             string             `gorm:"size:100" json:"license"`
	Reusable            bool               `gorm:"not null;default:false" json:"reusable"`
	Categories          []string           `gorm:"-" json:"-"`
	CategoryScores      map[string]float64 `gorm:"-" json:"-"`
}

type RecommendationCategory struct {
	VideoID    uuid.UUID        `gorm:"type:uuid;primaryKey;index:idx_recommendation_category_lookup,priority:3" json:"video_id"`
	Category   string           `gorm:"size:80;primaryKey;index:idx_recommendation_category_lookup,priority:2" json:"category"`
	Language   string           `gorm:"size:16;not null;index:idx_recommendation_category_lookup,priority:1" json:"language"`
	ViralScore float64          `gorm:"not null;default:0;index:idx_recommendation_category_lookup,priority:4,sort:desc" json:"viral_score"`
	Video      RecommendedVideo `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type ChannelMetricSnapshot struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID   string    `gorm:"size:64;not null;uniqueIndex:idx_channel_snapshot" json:"channel_id"`
	Subscribers int64     `gorm:"not null;default:0" json:"subscribers"`
	CapturedAt  time.Time `gorm:"not null;uniqueIndex:idx_channel_snapshot;index" json:"captured_at"`
}

func (snapshot *ChannelMetricSnapshot) EnsureID() {
	if snapshot.ID == uuid.Nil {
		snapshot.ID = uuid.New()
	}
}
