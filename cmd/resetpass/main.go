package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run ./cmd/resetpass <username> <new_password>")
		os.Exit(1)
	}
	username, password := os.Args[1], os.Args[2]

	db, err := sql.Open("sqlite", "./data/qacapsule.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	res, err := db.Exec(
		"UPDATE users SET password_hash = ?, require_password_change = 1 WHERE username = ?",
		string(hashed), username,
	)
	if err != nil {
		log.Fatal(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		log.Fatalf("user %q not found", username)
	}
	fmt.Printf("Password reset for %q (change required on next login).\n", username)
}
