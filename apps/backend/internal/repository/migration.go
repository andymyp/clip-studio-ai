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
		if err := prepareAuthenticationMigration(tx); err != nil {
			return err
		}
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

func prepareAuthenticationMigration(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.User{}) ||
		db.Migrator().HasColumn(&model.User{}, "PasswordHash") {
		return nil
	}

	if err := db.Exec(
		"ALTER TABLE users ADD COLUMN password_hash varchar(60)",
	).Error; err != nil {
		return err
	}
	// Existing foundation-era users never had credentials. This deliberately
	// invalid bcrypt value keeps those accounts locked until a future password
	// reset flow is introduced.
	if err := db.Exec(
		"UPDATE users SET password_hash = ? WHERE password_hash IS NULL",
		"!legacy-account-password-reset-required!",
	).Error; err != nil {
		return err
	}
	return db.Exec(
		"ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL",
	).Error
}
