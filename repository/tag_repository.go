// tag_repository.go
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/models"

	"github.com/google/uuid"
)

type TagRepository struct {
	DB *sql.DB
}

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{DB: db}
}

func (r *TagRepository) CreateTag(ctx context.Context, name string) (*models.Tag, error) {
	id := uuid.New()
	now := time.Now()
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO tags (id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`, id, name, now, now)
	if err != nil {
		return nil, err
	}
	return &models.Tag{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *TagRepository) GetTags(ctx context.Context) ([]models.Tag, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, created_at, updated_at FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (r *TagRepository) GetTagByID(ctx context.Context, id uuid.UUID) (*models.Tag, error) {
	var tag models.Tag
	err := r.DB.QueryRowContext(ctx, `SELECT id, name, created_at, updated_at FROM tags WHERE id = $1`, id).
		Scan(&tag.ID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *TagRepository) UpdateTag(ctx context.Context, id uuid.UUID, name string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE tags SET name = $1, updated_at = $2 WHERE id = $3
	`, name, time.Now(), id)
	return err
}

func (r *TagRepository) DeleteTag(ctx context.Context, id uuid.UUID) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM tags WHERE id = $1`, id)
	return err
}
