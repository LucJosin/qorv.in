package collection

import (
	"context"
	"fmt"
	"uuid"

	"github.com/lucjosin/qorv.in/internal/database"
	"github.com/lucjosin/qorv.in/internal/database/mariadb"
)

const selectCollectionsStmt = `
	SELECT
		c.id,
		c.workspace_id,
		c.name,
		c.description,
		c.category,
		c.is_public,
		c.color,
		c.icon,
		c.public_stats,
		c.pinned,
		c.deleted_at,
		c.created_at,
		c.updated_at
	FROM 
		collections c`

type MariaDBRepository struct {
	db *mariadb.Database
}

func NewMariaDBRepository(db *mariadb.Database) Repository {
	return &MariaDBRepository{
		db: db,
	}
}

func (r *MariaDBRepository) List(ctx context.Context, params Params) ([]Collection, error) {
	query := selectCollectionsStmt + ` WHERE c.deleted_at IS NULL`
	args := make([]any, 0, 5)
	if params.WorkspaceID != uuid.Nil() {
		query += ` AND c.workspace_id = ?`
		args = append(args, params.WorkspaceID.String())
	}
	if params.After != nil {
		query += ` AND (c.created_at < ? OR (c.created_at = ? AND c.id < ?))`
		args = append(args, params.After.CreatedAt, params.After.CreatedAt, params.After.ID.String())
	}
	query += ` ORDER BY c.created_at DESC, c.id DESC LIMIT ?`
	args = append(args, params.Limit+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer database.Close(ctx, rows)

	var entities []Collection
	for rows.Next() {
		entity, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning collection: %w", err)
		}
		entities = append(entities, entity)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return entities, nil
}
