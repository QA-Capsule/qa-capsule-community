package core

import (
	"database/sql"
	"log"
	"strings"
)

// tableHasColumn reports whether table has column (SQLite PRAGMA table_info).
func tableHasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column")
}

// execSchemaSQL runs idempotent DDL (CREATE IF NOT EXISTS, CREATE INDEX IF NOT EXISTS, etc.).
func execSchemaSQL(sqlStmt string) {
	if _, err := DB.Exec(sqlStmt); err != nil {
		log.Printf("[WARNING] Schema statement failed: %v", err)
	}
}

// addColumnIfNotExists applies ALTER TABLE ADD COLUMN only when the column is missing.
func addColumnIfNotExists(table, column, alterSQL string) {
	has, err := tableHasColumn(DB, table, column)
	if err != nil {
		log.Printf("[WARNING] Could not inspect %s.%s: %v", table, column, err)
		return
	}
	if has {
		return
	}
	if _, err := DB.Exec(alterSQL); err != nil {
		if isDuplicateColumnErr(err) {
			return
		}
		log.Printf("[WARNING] Could not add column %s.%s: %v", table, column, err)
	}
}
