package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
)

func Get(db *leveldb.DB, key string) string {
	value, err := db.Get([]byte(key), nil)
	if err != nil {
		log.Warnln(err)
		return ""
	}
	return string(value)
}

func Put(db *leveldb.DB, key, value string) {
	err := db.Put([]byte(key), []byte(value), nil)
	if err != nil {
		log.Warnln(err)
	}
}
