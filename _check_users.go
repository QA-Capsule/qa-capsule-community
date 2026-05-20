package main
import (
  "database/sql"
  "fmt"
  "log"
  _ "modernc.org/sqlite"
  "golang.org/x/crypto/bcrypt"
)
func main() {
  db, _ := sql.Open("sqlite", "./data/qacapsule.db")
  defer db.Close()
  rows, err := db.Query("SELECT username, role, is_active, require_password_change, substr(password_hash,1,20) FROM users")
  if err != nil { log.Fatal(err) }
  defer rows.Close()
  for rows.Next() {
    var u, role, ph string
    var active, req int
    rows.Scan(&u, &role, &active, &req, &ph)
    fmt.Printf("user=%s role=%s active=%d req_change=%d hash_prefix=%s\n", u, role, active, req, ph)
  }
  var hash string
  db.QueryRow("SELECT password_hash FROM users WHERE username='admin'").Scan(&hash)
  err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("admin"))
  fmt.Printf("admin/admin match: %v\n", err == nil)
}
