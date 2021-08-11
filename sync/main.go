package main

import (
	"github.com/mapprotocol/compass/libs/sync_cmd"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	sync_cmd.Run()

}
