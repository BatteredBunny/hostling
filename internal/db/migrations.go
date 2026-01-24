package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ariga.io/atlas/sql/migrate"
	"gorm.io/gorm"
)

type AtlasSchemaRevision struct {
	Version         string    `gorm:"primaryKey;column:version"`
	Description     string    `gorm:"column:description"`
	Type            uint      `gorm:"column:type"`
	Applied         int       `gorm:"column:applied"`
	Total           int       `gorm:"column:total"`
	ExecutedAt      time.Time `gorm:"column:executed_at"`
	ExecutionTime   int64     `gorm:"column:execution_time"` // Duration in nanoseconds
	Error           string    `gorm:"column:error"`
	ErrorStmt       string    `gorm:"column:error_stmt"`
	Hash            string    `gorm:"column:hash"`
	PartialHashes   string    `gorm:"column:partial_hashes"` // JSON-encoded []string
	OperatorVersion string    `gorm:"column:operator_version"`
}

func (AtlasSchemaRevision) TableName() string {
	return "atlas_schema_revisions"
}

type GormRevisionReadWriter struct {
	db *gorm.DB
}

func (g *GormRevisionReadWriter) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{
		Name:   "atlas_schema_revisions",
		Schema: "",
	}
}

func (g *GormRevisionReadWriter) ReadRevisions(ctx context.Context) (revisions []*migrate.Revision, err error) {
	var dbRevisions []AtlasSchemaRevision
	if err = g.db.WithContext(ctx).Order("applied ASC").Find(&dbRevisions).Error; err != nil {
		err = fmt.Errorf("failed to read revisions: %w", err)
		return
	}

	for _, dbRev := range dbRevisions {
		var rev *migrate.Revision
		rev, err = dbRevisionToMigrate(&dbRev)
		if err != nil {
			err = fmt.Errorf("failed to convert revision %s: %w", dbRev.Version, err)
			return
		}
		revisions = append(revisions, rev)
	}

	return
}

func (g *GormRevisionReadWriter) ReadRevision(ctx context.Context, version string) (rev *migrate.Revision, err error) {
	var dbRev AtlasSchemaRevision
	if err = g.db.WithContext(ctx).Where("version = ?", version).First(&dbRev).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			err = migrate.ErrRevisionNotExist
			return
		}

		err = fmt.Errorf("failed to read revision %s: %w", version, err)
		return
	}

	return dbRevisionToMigrate(&dbRev)
}

func (g *GormRevisionReadWriter) WriteRevision(ctx context.Context, rev *migrate.Revision) (err error) {
	dbRev, err := migrateToDBRevision(rev)
	if err != nil {
		return fmt.Errorf("failed to convert revision: %w", err)
	}

	if err := g.db.WithContext(ctx).Save(dbRev).Error; err != nil {
		return fmt.Errorf("failed to write revision %s: %w", rev.Version, err)
	}

	return
}

func (g *GormRevisionReadWriter) DeleteRevision(ctx context.Context, version string) (err error) {
	if err = g.db.WithContext(ctx).Where("version = ?", version).Delete(&AtlasSchemaRevision{}).Error; err != nil {
		return fmt.Errorf("failed to delete revision %s: %w", version, err)
	}
	return
}

func dbRevisionToMigrate(dbRev *AtlasSchemaRevision) (rev *migrate.Revision, err error) {
	rev = &migrate.Revision{
		Version:         dbRev.Version,
		Description:     dbRev.Description,
		Type:            migrate.RevisionType(dbRev.Type),
		Applied:         dbRev.Applied,
		Total:           dbRev.Total,
		ExecutedAt:      dbRev.ExecutedAt,
		ExecutionTime:   time.Duration(dbRev.ExecutionTime),
		Error:           dbRev.Error,
		ErrorStmt:       dbRev.ErrorStmt,
		Hash:            dbRev.Hash,
		OperatorVersion: dbRev.OperatorVersion,
	}

	if dbRev.PartialHashes != "" {
		var partialHashes []string
		if err = json.Unmarshal([]byte(dbRev.PartialHashes), &partialHashes); err != nil {
			err = fmt.Errorf("failed to unmarshal partial_hashes: %w", err)
			return
		}
		rev.PartialHashes = partialHashes
	}

	return
}

func migrateToDBRevision(rev *migrate.Revision) (dbRev *AtlasSchemaRevision, err error) {
	dbRev = &AtlasSchemaRevision{
		Version:         rev.Version,
		Description:     rev.Description,
		Type:            uint(rev.Type),
		Applied:         rev.Applied,
		Total:           rev.Total,
		ExecutedAt:      rev.ExecutedAt,
		ExecutionTime:   int64(rev.ExecutionTime),
		Error:           rev.Error,
		ErrorStmt:       rev.ErrorStmt,
		Hash:            rev.Hash,
		OperatorVersion: rev.OperatorVersion,
	}

	if len(rev.PartialHashes) > 0 {
		var partialHashesJSON []byte
		partialHashesJSON, err = json.Marshal(rev.PartialHashes)
		if err != nil {
			err = fmt.Errorf("failed to marshal partial_hashes: %w", err)
			return
		}
		dbRev.PartialHashes = string(partialHashesJSON)
	}

	return
}

func NewRevisionReaderWriter(db *gorm.DB) migrate.RevisionReadWriter {
	return &GormRevisionReadWriter{db: db}
}
