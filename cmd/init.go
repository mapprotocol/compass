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

var levelDbInstance *leveldb.DB

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
}
func initDb() {
	if _, err := os.Stat(runtimePath); os.IsNotExist(err) {
		err := os.Mkdir(runtimePath, 0700)
		if err != nil {
			log.Fatal("make runtime dir error")
		}
	}
	var err error
	levelDbInstance, err = leveldb.OpenFile(filepath.Join(runtimePath, "db"), nil)
	if err != nil {
		log.Fatal("open levelDbInstance file error :", err)
	}
}
