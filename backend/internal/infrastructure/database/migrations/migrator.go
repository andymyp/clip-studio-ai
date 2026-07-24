package migrations

import (
	"fmt"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/domain/model"
	"gorm.io/gorm"
)

// Up applies additive schema changes in dependency order. GORM records no
// migration history, so destructive changes must be introduced as explicit,
// versioned migrations in future releases.
func Up(db *gorm.DB) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		return tx.AutoMigrate(
			&model.User{},
			&model.Video{},
			&model.Clip{},
			&model.AnalysisJob{},
			&model.RenderJob{},
			&model.Subtitle{},
			&model.Watermark{},
		)
	})
	if err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}
