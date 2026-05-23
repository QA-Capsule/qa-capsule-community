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
	requireChange := 1
	args := os.Args[1:]
	if len(args) >= 1 && (args[0] == "--no-change-required" || args[0] == "-n") {
		requireChange = 0
		args = args[1:]
	}
	if len(args) != 2 {
		fmt.Println("Usage: go run ./cmd/resetpass [--no-change-required] <username> <new_password>")
		os.Exit(1)
	}
	username, password := args[0], args[1]

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
		"UPDATE users SET password_hash = ?, require_password_change = ?, is_active = 1 WHERE username = ?",
		string(hashed), requireChange, username,
	)
	if err != nil {
		log.Fatal(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		log.Fatalf("user %q not found", username)
	}
	if requireChange == 1 {
		fmt.Printf("Password reset for %q (must change password on next login).\n", username)
	} else {
		fmt.Printf("Password reset for %q (ready to sign in at http://localhost:9000).\n", username)
	}
}
