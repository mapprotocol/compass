package libs

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

func WriteLog(s string) {
	w := bufio.NewWriter(SignLogFile)
	_, err := fmt.Fprintln(w, s)
	if err != nil {
		log.Infoln("Write log error", err)
		return
	}
	err = w.Flush()
	if err != nil {
		log.Infoln("Write log error", err)
		return
	}
}
func GetLastLineWithSeek() string {
	fileHandle, err := os.Open(LogFile)

	if err != nil {
		panic("Cannot open file")
	}
	defer func(fileHandle *os.File) {
		err := fileHandle.Close()
		if err != nil {
			log.Infoln(err)
		}
	}(fileHandle)

	line := ""
	var cursor int64 = 0
	stat, _ := fileHandle.Stat()
	filesize := stat.Size()
	if filesize == 0 {
		return line
	}
	for {
		cursor -= 1
		_, err := fileHandle.Seek(cursor, io.SeekEnd)
		if err != nil {
			return ""
		}

		char := make([]byte, 1)
		_, err = fileHandle.Read(char)
		if err != nil {
			return ""
		}

		if cursor != -1 && (char[0] == 10 || char[0] == 13) { // stop if we find a line
			break
		}

		line = fmt.Sprintf("%s%s", string(char), line) // there is more efficient way

		if cursor == -filesize { // stop if we are at the begining
			break
		}
	}

	return line
}

func WriteConfig(key string, value string) {
	err := DiskCache.Write(key, []byte(value))
	if err != nil {
		return
	}
}
func EraseConfig(key string) {
	err := DiskCache.Erase(key)
	if err != nil {
		return
	}
}
func ReadConfig(key string, defaultValue string) string {
	b, err := DiskCache.Read(key)
	if err == nil {
		return string(b)
	} else {
		return defaultValue
	}
}
func ReadConfigWithCondition(key string, defaultValue string, f func(string) bool) string {
	b, err := DiskCache.Read(key)

	if err == nil && f(string(b)) {
		return string(b)
	} else {
		return defaultValue
	}
}
