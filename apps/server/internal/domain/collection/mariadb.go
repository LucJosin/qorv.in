package collection

import (
	"context"
	"fmt"
	"strings"

	"github.com/lucjosin/qorv.in/internal/database"
	"github.com/lucjosin/qorv.in/internal/database/mariadb"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/slogx"
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

func (r *MariaDBRepository) List(ctx context.Context, params ListParams) ([]Collection, error) {
	log := slogx.FromCtx(ctx)

	var args []any
	query := selectCollectionsStmt + ` WHERE c.deleted_at IS NULL`
	if params.Filters.WorkspaceID != nil {
		query += ` AND c.workspace_id = ?`
		args = append(args, params.Filters.WorkspaceID.String())
	}
	if params.Filters.Name != nil {
		escaper := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")
		name := escaper.Replace(*params.Filters.Name)
		query += ` AND c.name LIKE ? ESCAPE '!'`
		args = append(args, "%"+name+"%")
	}
	if params.Filters.Category != nil {
		query += ` AND c.category = ?`
		args = append(args, *params.Filters.Category)
	}
	if params.Filters.IsPublic != nil {
		query += ` AND c.is_public = ?`
		args = append(args, *params.Filters.IsPublic)
	}
	if params.Filters.Pinned != nil {
		query += ` AND c.pinned = ?`
		args = append(args, *params.Filters.Pinned)
	}
	if params.Filters.PublicStats != nil {
		query += ` AND c.public_stats = ?`
		args = append(args, *params.Filters.PublicStats)
	}
	if params.After != nil {
		if params.Sort.Order == "asc" {
			query += ` AND (c.created_at > ? OR (c.created_at = ? AND c.id > ?))`
		} else {
			query += ` AND (c.created_at < ? OR (c.created_at = ? AND c.id < ?))`
		}
		args = append(args, params.After.CreatedAt, params.After.CreatedAt, params.After.ID.String())
	}
	if params.Sort.Order == "asc" {
		query += ` ORDER BY c.created_at ASC, c.id ASC LIMIT ?`
	} else {
		query += ` ORDER BY c.created_at DESC, c.id DESC LIMIT ?`
	}
	args = append(args, params.Limit+1)

	log.Trace("querying collections", "sql", query)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mariadb.ParseError(err, "querying database")
	}
	defer database.Close(ctx, rows)

	var entities []Collection
	for rows.Next() {
		entity, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: scanning collection: %v", errs.ErrInternal, err)
		}
		entities = append(entities, entity)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return entities, nil
}
