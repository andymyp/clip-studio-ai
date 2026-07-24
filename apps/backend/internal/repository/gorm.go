package repository

import (
	"context"
	"errors"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type baseRepository[T any] struct {
	db *gorm.DB
}

func (repo *baseRepository[T]) Create(ctx context.Context, entity *T) error {
	if identifiable, ok := any(entity).(interface{ EnsureID() }); ok {
		identifiable.EnsureID()
	}
	return translateError(repo.db.WithContext(ctx).Omit(clause.Associations).Create(entity).Error)
}

func (repo *baseRepository[T]) getByID(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	if err := repo.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, translateError(err)
	}
	return &entity, nil
}

func (repo *baseRepository[T]) Update(ctx context.Context, entity *T) error {
	result := repo.db.WithContext(ctx).
		Model(entity).
		Select("*").
		Omit("created_at", clause.Associations).
		Updates(entity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (repo *baseRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	var entity T
	result := repo.db.WithContext(ctx).Delete(&entity, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (repo *baseRepository[T]) listBy(
	ctx context.Context,
	column string,
	id uuid.UUID,
	limit int,
	offset int,
) ([]T, error) {
	var entities []T
	query := repo.db.WithContext(ctx).
		Where(column+" = ?", id).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrConflict
	}
	return err
}

type UserRepository struct{ *baseRepository[model.User] }

func (repo *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return repo.getByID(ctx, id)
}

func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := repo.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, translateError(err)
	}
	return &user, nil
}

type VideoRepository struct{ *baseRepository[model.Video] }

func (repo *VideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Video, error) {
	return repo.getByID(ctx, id)
}

func (repo *VideoRepository) GetByIDForUser(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (*model.Video, error) {
	var video model.Video
	if err := repo.db.WithContext(ctx).
		First(&video, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, translateError(err)
	}
	return &video, nil
}

func (repo *VideoRepository) Search(
	ctx context.Context,
	userID uuid.UUID,
	keyword string,
	limit int,
) ([]model.Video, error) {
	var videos []model.Video
	pattern := "%" + keyword + "%"
	err := repo.db.WithContext(ctx).
		Where("user_id = ? AND (title ILIKE ? OR description ILIKE ?)", userID, pattern, pattern).
		Order("views DESC, created_at DESC").
		Limit(limit).
		Find(&videos).Error
	return videos, err
}

func (repo *VideoRepository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	offset int,
) ([]model.Video, error) {
	return repo.listBy(ctx, "user_id", userID, limit, offset)
}

type ClipRepository struct{ *baseRepository[model.Clip] }

func (repo *ClipRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Clip, error) {
	return repo.getByID(ctx, id)
}

func (repo *ClipRepository) ListByVideoID(
	ctx context.Context,
	videoID uuid.UUID,
) ([]model.Clip, error) {
	return repo.listBy(ctx, "video_id", videoID, 0, 0)
}

type AnalysisJobRepository struct {
	*baseRepository[model.AnalysisJob]
}

func (repo *AnalysisJobRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.AnalysisJob, error) {
	return repo.getByID(ctx, id)
}

func (repo *AnalysisJobRepository) GetByIDForUser(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (*model.AnalysisJob, error) {
	var job model.AnalysisJob
	err := repo.db.WithContext(ctx).
		Joins("JOIN videos ON videos.id = analysis_jobs.video_id").
		Where("analysis_jobs.id = ? AND videos.user_id = ?", id, userID).
		First(&job).Error
	if err != nil {
		return nil, translateError(err)
	}
	return &job, nil
}

func (repo *AnalysisJobRepository) ListByVideoID(
	ctx context.Context,
	videoID uuid.UUID,
) ([]model.AnalysisJob, error) {
	return repo.listBy(ctx, "video_id", videoID, 0, 0)
}

type RenderJobRepository struct {
	*baseRepository[model.RenderJob]
}

func (repo *RenderJobRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.RenderJob, error) {
	return repo.getByID(ctx, id)
}

func (repo *RenderJobRepository) ListByClipID(
	ctx context.Context,
	clipID uuid.UUID,
) ([]model.RenderJob, error) {
	return repo.listBy(ctx, "clip_id", clipID, 0, 0)
}

type SubtitleRepository struct {
	*baseRepository[model.Subtitle]
}

func (repo *SubtitleRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.Subtitle, error) {
	return repo.getByID(ctx, id)
}

func (repo *SubtitleRepository) ListByVideoID(
	ctx context.Context,
	videoID uuid.UUID,
) ([]model.Subtitle, error) {
	return repo.listBy(ctx, "video_id", videoID, 0, 0)
}

type WatermarkRepository struct {
	*baseRepository[model.Watermark]
}

func (repo *WatermarkRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.Watermark, error) {
	return repo.getByID(ctx, id)
}

func (repo *WatermarkRepository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]model.Watermark, error) {
	return repo.listBy(ctx, "user_id", userID, 0, 0)
}

type Repositories struct {
	Users        UserRepositoryContract
	Videos       VideoRepositoryContract
	Clips        ClipRepositoryContract
	AnalysisJobs AnalysisJobRepositoryContract
	RenderJobs   RenderJobRepositoryContract
	Subtitles    SubtitleRepositoryContract
	Watermarks   WatermarkRepositoryContract
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{
		Users: &UserRepository{
			baseRepository: &baseRepository[model.User]{db: db},
		},
		Videos: &VideoRepository{
			baseRepository: &baseRepository[model.Video]{db: db},
		},
		Clips: &ClipRepository{
			baseRepository: &baseRepository[model.Clip]{db: db},
		},
		AnalysisJobs: &AnalysisJobRepository{
			baseRepository: &baseRepository[model.AnalysisJob]{db: db},
		},
		RenderJobs: &RenderJobRepository{
			baseRepository: &baseRepository[model.RenderJob]{db: db},
		},
		Subtitles: &SubtitleRepository{
			baseRepository: &baseRepository[model.Subtitle]{db: db},
		},
		Watermarks: &WatermarkRepository{
			baseRepository: &baseRepository[model.Watermark]{db: db},
		},
	}
}
