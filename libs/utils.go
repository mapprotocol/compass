package libs

import (
	"bufio"
	"fmt"
	"github.com/ethereum/go-ethereum/params"
	"io"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"
)

func fileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

func readString() string {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		log.Fatal(err)
	}
	return input
}
func WriteLog(s string) {
	w := bufio.NewWriter(SignLogFile)
	_, err := fmt.Fprintln(w, s)
	if err != nil {
		log.Println("Write log error", err)
		return
	}
	err = w.Flush()
	if err != nil {
		log.Println("Write log error", err)
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
			log.Println(err)
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
func WeiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}

func Unix2Time(timestamp big.Int) time.Time {
	i, err := strconv.ParseInt(timestamp.String(), 10, 64)
	if err != nil {
		return time.Unix(0, 0)
	}
	return time.Unix(i, 0)
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
	if err == nil && f(key) {
		return string(b)
	} else {
		return defaultValue
	}
}
