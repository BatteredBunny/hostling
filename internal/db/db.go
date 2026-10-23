package db

import (
	"context"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

// WithContext returns an independent session; the shared database is unchanged.
// Typed queries pass its Statement.Context to terminal operations as well.
func (db Database) WithContext(ctx context.Context) *Database {
	return &Database{DB: db.DB.WithContext(ctx)}
}
