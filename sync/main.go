package main

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"signmap/libs"
	"strconv"
	"time"
)

var (
	srcClient                   *ethclient.Client
	dstClient                   *ethclient.Client
	db                          *sql.DB
	lastSyncNum                 uint64 = 0
	dstChainId                  *big.Int
	privateKey                  *ecdsa.PrivateKey
	fromAddress                 common.Address
	gasPrice                    *big.Int
	waitReceiptPerTime          = 3 * time.Second
	waitReceiptTimes            = 100
	syncBlockAfterMinuteInChain uint64
)

func main() {
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)
	num, err := srcClient.BlockNumber(context.Background())
	go func() {
		gasPrice, err = dstClient.SuggestGasPrice(context.Background())
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Minute)
	}()
	for {
		if num <= lastSyncNum {
			time.Sleep(time.Minute)
			num, err = srcClient.BlockNumber(context.Background())
			if err != nil {
				log.Println(nil)
			}
			continue
		}
		if !syncBlock(lastSyncNum) {
			time.Sleep(time.Minute)
		} else {
			lastSyncNum += 1
		}
	}
}
func syncBlock(num uint64) bool {
	block, err := srcClient.BlockByNumber(context.Background(), big.NewInt(int64(num)))
	if err != nil {
		return false
	}
	if block.Time()+syncBlockAfterMinuteInChain*60 > uint64(time.Now().Unix()) {
		fmt.Printf("Current under consideration block %d , it can be  synchronized after %d seconds ", num, block.Time()+syncBlockAfterMinuteInChain*60-uint64(time.Now().Unix()))
		println()
		return false
	}

	data, _ := block.Header().MarshalJSON()
	//todo Cut out the data

	nonce, err := dstClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Println(err)
	}

	msg := ethereum.CallMsg{From: fromAddress, To: &fromAddress, GasPrice: gasPrice, Data: data}
	gas, _ := dstClient.EstimateGas(context.Background(), msg)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		Value:    nil,
		To:       &fromAddress,
		GasPrice: gasPrice,
		Gas:      gas,
		Data:     data,
	})
	if err != nil {
		log.Println(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(dstChainId), privateKey)
	if err != nil {
		log.Println(err)
	}

	err = dstClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("tx sent: %s", signedTx.Hash().Hex())
	println()
	stmt, _ := db.Prepare("insert into block (id,hash,tx_hash,info) values (?,?,?,?)")
	stmt.Exec(num, block.Hash().Hex(), signedTx.Hash().Hex(), "")
	go func() {
		time.Sleep(waitReceiptPerTime)
		var i = 0
		for {
			receipt, err := dstClient.TransactionReceipt(context.Background(), signedTx.Hash())
			if err != nil && i < waitReceiptTimes {
				time.Sleep(waitReceiptPerTime)
				i += 1
				continue
			}

			if err == nil || i == waitReceiptTimes {
				stmt, _ = db.Prepare("update block set info = ? where id = ?")
				_, err := stmt.Exec("error", num)
				if err != nil {
					log.Println(err)
				}
				return
			} else {
				switch receipt.Status {
				case types.ReceiptStatusSuccessful:
					stmt, _ = db.Prepare("update block set info = ? where id = ?")
					_, err := stmt.Exec("ok", num)
					if err != nil {
						log.Println(err)
					}
					return
				case types.ReceiptStatusFailed:
					stmt, _ = db.Prepare("update block set info = ? where id = ?")
					_, err := stmt.Exec("error", num)
					if err != nil {
						log.Println(err)
					}
					return
				default:
					//should unreachable
					log.Println("Unknown receipt status: ", receipt.Status)
					time.Sleep(waitReceiptPerTime)
					i += 1
					continue
				}
			}
		}
	}()
	return true
}

func GetKey() {
	path := os.Getenv("keystore")
	password := os.Getenv("password")

	if !common.FileExist(path) {
		log.Fatal("Keystore file not exist, config at .env")
	}
	keyJson, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	key, err := keystore.DecryptKey(keyJson, password)
	if err != nil {
		log.Fatal("Incorrect password! config at .env")
	}
	privateKey = key.PrivateKey
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	fromAddress = crypto.PubkeyToAddress(*publicKeyECDSA)
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	srcRpcUrl := os.Getenv("src_rpc_url")
	srcClient = libs.GetClientByUrl(srcRpcUrl)
	if srcClient == nil {
		log.Fatal("Source blockchain can not connect, config at .env file")
	}
	dstRpcUrl := os.Getenv("dst_rpc_url")
	if syncBlockAfterMinuteInChainValue, err := strconv.Atoi(os.Getenv("sync_block_after_minute_in_chain")); err != nil {
		syncBlockAfterMinuteInChain = 10
	} else {
		syncBlockAfterMinuteInChain = uint64(syncBlockAfterMinuteInChainValue)
	}
	dstClient = libs.GetClientByUrl(dstRpcUrl)
	if dstClient == nil {
		log.Fatal("Destination blockchain can not connect, config at .env file")
	}
	dstChainId, _ = dstClient.NetworkID(context.Background())
	GetKey()
	initDb()
}
func initDb() {
	dbFile := os.Getenv("sqlite_db_file")
	var err error
	if common.FileExist(dbFile) {
		db, err = sql.Open("sqlite3", dbFile)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		db, err = sql.Open("sqlite3", dbFile)
		if err != nil {
			log.Fatal(err)
		}
		db.Exec("create table block (id bigint,hash text,tx_hash text, info text)")
	}
	db.QueryRow("select id from block order by id desc limit 1").Scan(&lastSyncNum)
	lastSyncNum += 1
}
