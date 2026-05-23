package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	path := "./data/qacapsule.db"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT username, role, is_active, require_password_change FROM users ORDER BY username`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("users in", path)
	for rows.Next() {
		var u, role string
		var active, req int
		rows.Scan(&u, &role, &active, &req)
		fmt.Printf("  %-35s role=%-8s active=%d change_required=%d\n", u, role, active, req)
	}
}
