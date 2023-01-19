package monitor

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
)

func InitSql() {
	if db != nil {
		return
	}
	conn := os.Getenv("bridgeConn")
	tmpDb, err := sql.Open("mysql", conn)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}
	db = tmpDb
}
