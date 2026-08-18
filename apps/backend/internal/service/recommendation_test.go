package service

import (
	"strings"
	"testing"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

func TestPercentileViralScoreRewardsVelocityAndEngagement(t *testing.T) {
	now := time.Now().UTC()
	videos := []model.RecommendedVideo{
		{
			ExternalID: "strong", Language: "en", Categories: []string{"podcast"},
			Views: 100_000, Likes: 8_000, Comments: 800,
			Subscribers: 10_000, PublishedAt: now.Add(-2 * time.Hour),
		},
		{
			ExternalID: "weak", Language: "en", Categories: []string{"podcast"},
			Views: 1_000, Likes: 10, Comments: 1,
			Subscribers: 10_000, PublishedAt: now.Add(-20 * time.Hour),
		},
	}
	scoreRecommendations(videos, map[string]float64{}, now)
	strongScore := videos[0].CategoryScores["podcast"]
	weakScore := videos[1].CategoryScores["podcast"]
	if strongScore <= weakScore {
		t.Fatalf("strong score %v must exceed weak score %v", strongScore, weakScore)
	}
	if strongScore < 0 || strongScore > 100 {
		t.Fatalf("viral score %v is outside 0..100", strongScore)
	}
}

func TestNormalizeLanguageUsesBaseLanguage(t *testing.T) {
	if got := normalizeLanguage("EN-us"); got != "en" {
		t.Fatalf("normalizeLanguage() = %q, want en", got)
	}
}

func TestDeduplicateRecommendationsUsesExternalID(t *testing.T) {
	videos := deduplicateRecommendations([]model.RecommendedVideo{
		{ExternalID: "video-1", Categories: []string{"podcast"}, Views: 10},
		{ExternalID: "video-1", Categories: []string{"interview"}, Views: 20},
		{ExternalID: " ", Category: "story"},
	})

	if len(videos) != 1 {
		t.Fatalf("deduplicateRecommendations() returned %d videos, want 1", len(videos))
	}
	if videos[0].Views != 20 || len(videos[0].Categories) != 2 {
		t.Fatalf("deduplicateRecommendations() = %#v", videos[0])
	}
}

func TestRecommendationStyleCategories(t *testing.T) {
	gameplay := recommendationStyleCategories("gameplay", "trending")
	if len(gameplay) != 4 || gameplay[0] != "gameplay" {
		t.Fatalf("gameplay categories = %#v", gameplay)
	}

	auto := recommendationStyleCategories("auto", "science")
	if len(auto) != 1 || auto[0] != "science" {
		t.Fatalf("auto categories = %#v", auto)
	}
}

func TestRecommendationStyleConditionIncludesMetadataSignals(t *testing.T) {
	condition, arguments := recommendationStyleCondition(
		recommendationStyleCategories("gameplay", "trending"),
		recommendationStyleTerms("gameplay"),
		recommendationStyleCategoryIDs("gameplay"),
	)

	if !strings.Contains(condition, "recommended_videos.title") {
		t.Fatalf("condition does not search video metadata: %s", condition)
	}
	if !strings.Contains(condition, "category_id IN") {
		t.Fatalf("condition does not search YouTube category ID: %s", condition)
	}
	if len(arguments) != 7 {
		t.Fatalf("arguments count = %d, want 7", len(arguments))
	}
}
