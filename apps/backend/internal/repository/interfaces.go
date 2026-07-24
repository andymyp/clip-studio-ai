package repository

import (
	"context"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/google/uuid"
)

type UserRepositoryContract interface {
	Create(context.Context, *model.User) error
	GetByID(context.Context, uuid.UUID) (*model.User, error)
	GetByEmail(context.Context, string) (*model.User, error)
	Update(context.Context, *model.User) error
	Delete(context.Context, uuid.UUID) error
}

type VideoRepositoryContract interface {
	Create(context.Context, *model.Video) error
	GetByID(context.Context, uuid.UUID) (*model.Video, error)
	GetByIDForUser(context.Context, uuid.UUID, uuid.UUID) (*model.Video, error)
	Search(context.Context, uuid.UUID, string, int) ([]model.Video, error)
	ListByUserID(context.Context, uuid.UUID, int, int) ([]model.Video, error)
	Update(context.Context, *model.Video) error
	Delete(context.Context, uuid.UUID) error
}

type ClipRepositoryContract interface {
	Create(context.Context, *model.Clip) error
	GetByID(context.Context, uuid.UUID) (*model.Clip, error)
	ListByVideoID(context.Context, uuid.UUID) ([]model.Clip, error)
	Update(context.Context, *model.Clip) error
	Delete(context.Context, uuid.UUID) error
}

type AnalysisJobRepositoryContract interface {
	Create(context.Context, *model.AnalysisJob) error
	GetByID(context.Context, uuid.UUID) (*model.AnalysisJob, error)
	GetByIDForUser(context.Context, uuid.UUID, uuid.UUID) (*model.AnalysisJob, error)
	ListByVideoID(context.Context, uuid.UUID) ([]model.AnalysisJob, error)
	ListByUserID(context.Context, uuid.UUID, int) ([]model.AnalysisJob, error)
	Update(context.Context, *model.AnalysisJob) error
	Delete(context.Context, uuid.UUID) error
}

type RenderJobRepositoryContract interface {
	Create(context.Context, *model.RenderJob) error
	GetByID(context.Context, uuid.UUID) (*model.RenderJob, error)
	ListByClipID(context.Context, uuid.UUID) ([]model.RenderJob, error)
	Update(context.Context, *model.RenderJob) error
	Delete(context.Context, uuid.UUID) error
}

type SubtitleRepositoryContract interface {
	Create(context.Context, *model.Subtitle) error
	GetByID(context.Context, uuid.UUID) (*model.Subtitle, error)
	ListByVideoID(context.Context, uuid.UUID) ([]model.Subtitle, error)
	Update(context.Context, *model.Subtitle) error
	Delete(context.Context, uuid.UUID) error
}

type WatermarkRepositoryContract interface {
	Create(context.Context, *model.Watermark) error
	GetByID(context.Context, uuid.UUID) (*model.Watermark, error)
	ListByUserID(context.Context, uuid.UUID) ([]model.Watermark, error)
	Update(context.Context, *model.Watermark) error
	Delete(context.Context, uuid.UUID) error
}
