package model

import "time"

type VideoSearchResult struct {
	ExternalID      string    `json:"external_id"`
	Platform        string    `json:"platform"`
	CategoryID      string    `json:"category_id"`
	Title           string    `json:"title"`
	ChannelID       string    `json:"channel_id"`
	ChannelTitle    string    `json:"channel_title"`
	YouTubeUsername string    `json:"youtube_username"`
	Description     string    `json:"-"`
	Language        string    `json:"-"`
	Subscribers     int64     `json:"-"`
	PublishedAt     time.Time `json:"-"`
	URL             string    `json:"url"`
	EmbedURL        string    `json:"embed_url"`
	MediaURL        string    `json:"media_url"`
	Thumbnail       string    `json:"thumbnail"`
	Duration        float64   `json:"duration"`
	Views           int64     `json:"views"`
	Likes           int64     `json:"likes"`
	Comments        int64     `json:"comments"`
	Score           int64     `json:"score"`
	License         string    `json:"license"`
	Reusable        bool      `json:"reusable"`
}
