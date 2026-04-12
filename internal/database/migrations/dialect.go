package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

func isPostgres(db *gorm.DB) bool {
	return db.Dialector.Name() == "postgres"
}

func indexExists(db *gorm.DB, tableName, indexName string) (bool, error) {
	if isPostgres(db) {
		var n int64
		err := db.Raw(
			`SELECT COUNT(*) FROM pg_indexes WHERE schemaname = CURRENT_SCHEMA() AND tablename = ? AND indexname = ?`,
			tableName, indexName,
		).Scan(&n).Error
		return n > 0, err
	}
	var n int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`,
		tableName, indexName,
	).Scan(&n).Error
	return n > 0, err
}

func fkConstraintExists(db *gorm.DB, tableName, constraintName string) (bool, error) {
	if isPostgres(db) {
		var n int64
		err := db.Raw(
			`SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = CURRENT_SCHEMA() AND table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'`,
			tableName, constraintName,
		).Scan(&n).Error
		return n > 0, err
	}
	var n int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'`,
		tableName, constraintName,
	).Scan(&n).Error
	return n > 0, err
}

func tableExists(db *gorm.DB, tableName string) (bool, error) {
	if isPostgres(db) {
		var n int64
		err := db.Raw(
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?`,
			tableName,
		).Scan(&n).Error
		return n > 0, err
	}
	var n int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`,
		tableName,
	).Scan(&n).Error
	return n > 0, err
}

func dropForeignKeyBestEffort(db *gorm.DB, tableName, constraintName string) {
	if isPostgres(db) {
		_ = db.Exec(fmt.Sprintf(`ALTER TABLE %q DROP CONSTRAINT IF EXISTS %q`, tableName, constraintName)).Error
		return
	}
	_ = db.Exec(fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", tableName, constraintName)).Error
}

func dropIndexBestEffort(db *gorm.DB, tableName, indexName string) {
	if isPostgres(db) {
		_ = db.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %q`, indexName)).Error
		return
	}
	_ = db.Exec(fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", tableName, indexName)).Error
}
