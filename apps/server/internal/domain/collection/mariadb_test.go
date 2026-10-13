package collection_test

import (
	"context"
	"database/sql"
	"net"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
	"github.com/lucjosin/qorv.in/internal/pagination"

	driver "github.com/go-sql-driver/mysql"
)

// This opt-in test shadows collections with a connection-local temporary table.
func TestMariaDBRepository_CursorContinuation(t *testing.T) {
	if os.Getenv("QORVIN_DATABASE_TESTS") != "1" {
		t.Skip("set QORVIN_DATABASE_TESTS=1 and DATABASE_* to run MariaDB integration tests")
	}
	db := temporaryDatabase(t)
	ctx := context.Background()
	createdAt := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	workspaceID := uuid.NewV7()
	for i, id := range ids {
		timestamp := createdAt
		if i == 0 {
			timestamp = createdAt.Add(-time.Second)
		}
		insertCollection(t, db, id, workspaceID, timestamp)
	}
	insertCollection(t, db, uuid.NewV7(), uuid.NewV7(), createdAt.Add(time.Second))
	service := collection.NewService(collection.NewMariaDBRepository(db))
	params := collection.Params{
		Limit:       2,
		WorkspaceID: workspaceID,
	}
	first, err := service.List(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	got := pagination.Page[uuid.UUID]{
		Items:   collectionIDs(first.Items),
		HasMore: first.HasMore,
	}
	want := pagination.Page[uuid.UUID]{
		Items:   []uuid.UUID{ids[3], ids[2]},
		HasMore: true,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("first page mismatch (-want +got):\n%s", diff)
	}
	position := pagination.Cursor{
		CreatedAt: first.Items[1].CreatedAt,
		ID:        first.Items[1].ID,
	}
	_, err = db.ExecContext(ctx, "DELETE FROM collections WHERE id = ?", position.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	insertCollection(t, db, uuid.NewV7(), workspaceID, createdAt.Add(time.Second))
	params.After = &position
	last, err := service.List(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	got = pagination.Page[uuid.UUID]{
		Items:   collectionIDs(last.Items),
		HasMore: last.HasMore,
	}
	want = pagination.Page[uuid.UUID]{
		Items:   []uuid.UUID{ids[1], ids[0]},
		HasMore: false,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("last page mismatch (-want +got):\n%s", diff)
	}
	lastPosition := pagination.Cursor{
		CreatedAt: last.Items[1].CreatedAt,
		ID:        last.Items[1].ID,
	}
	params.After = &lastPosition
	empty, err := service.List(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	{
		got := empty
		want := pagination.Page[collection.Collection]{
			Items: []collection.Collection{},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("empty page mismatch (-want +got):\n%s", diff)
		}
	}
}

func temporaryDatabase(t *testing.T) *sql.DB {
	t.Helper()
	config := driver.NewConfig()
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_PORT"))
	config.User = os.Getenv("DATABASE_USER")
	config.Passwd = os.Getenv("DATABASE_PASSWORD")
	config.DBName = os.Getenv("DATABASE_NAME")
	config.ParseTime = true
	config.Timeout = 5 * time.Second
	config.ReadTimeout = 5 * time.Second
	config.WriteTimeout = 5 * time.Second
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		err := db.Close()
		if err != nil {
			t.Error(err)
		}
	})
	_, err = db.ExecContext(context.Background(), `CREATE TEMPORARY TABLE collections (
		id UUID PRIMARY KEY,
		workspace_id UUID NOT NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		category VARCHAR(50),
		is_public BOOLEAN NOT NULL DEFAULT FALSE,
		color VARCHAR(7),
		icon VARCHAR(100),
		public_stats BOOLEAN NOT NULL DEFAULT FALSE,
		pinned BOOLEAN NOT NULL DEFAULT FALSE,
		deleted_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		INDEX idx_collections_cursor (deleted_at, created_at, id)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func insertCollection(t *testing.T, db *sql.DB, id, workspaceID uuid.UUID, createdAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO collections
		(id, workspace_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id.String(), workspaceID.String(), "test", createdAt, createdAt)
	if err != nil {
		t.Fatal(err)
	}
}

func collectionIDs(items []collection.Collection) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
