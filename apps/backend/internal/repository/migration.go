package repository

import (
	"fmt"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Migrate applies additive schema changes in dependency order. GORM records no
// migration history, so destructive changes must be introduced as explicit,
// versioned migrations in future releases.
func Migrate(db *gorm.DB) error {
	quietDB := db.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	err := quietDB.Transaction(func(tx *gorm.DB) error {
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
