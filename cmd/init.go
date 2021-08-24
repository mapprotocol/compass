package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path/filepath"
)

const (
	runtimePath = "runtime"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	if _, err := os.Stat(runtimePath); os.IsNotExist(err) {
		err := os.Mkdir(runtimePath, 0700)
		if err != nil {
			log.Fatal("make runtime dir error")
		}
	}

	db, err := leveldb.OpenFile(filepath.Join(runtimePath, "db"), nil)
	if err != nil {
		log.Fatal("open db file error")
	}
	defer func(db *leveldb.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)
}
