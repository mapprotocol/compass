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
	"time"
)

var (
	srcClient   *ethclient.Client
	dstClient   *ethclient.Client
	db          *sql.DB
	lastSyncNum uint64 = 0
	dstChainId  *big.Int
	privateKey  *ecdsa.PrivateKey
	fromAddress common.Address
	gasPrice    *big.Int
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
		lastSyncNum += 1
		synBlock(lastSyncNum)
	}
}
func synBlock(num uint64) {
	block, err := srcClient.BlockByNumber(context.Background(), big.NewInt(int64(num)))
	if err != nil {
		return
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
	go func() {
		var i = 0
		for {
			receipt, err := dstClient.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil && i < 100 {
				time.Sleep(time.Minute)
				i += 1
				continue
			}
			stmt, _ := db.Prepare("insert into block (id,hash,info) values (?,?,?)")
			if err == nil || i == 100 {
				_, err := stmt.Exec(num, block.Hash().Hex(), "error")
				if err != nil {
					println(err)
				}
				return
			} else {
				switch receipt.Status {
				case types.ReceiptStatusSuccessful:
					_, err := stmt.Exec(num, block.Hash().Hex(), "ok")
					if err != nil {
						println(err)
					}
					return
				case types.ReceiptStatusFailed:
					_, err := stmt.Exec(num, block.Hash().Hex(), "error")
					if err != nil {
						println(err)
					}
					return
				default:
					//should unreachable
					log.Println("Unknown receipt status: ", receipt.Status)
					time.Sleep(time.Minute)
					i += 1
					continue
				}
			}
		}
	}()
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
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Println("error casting public key to ECDSA")
	}
	fromAddress = crypto.PubkeyToAddress(*publicKeyECDSA)
	println(fromAddress.Hex())
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	srcRpcUrl := os.Getenv("src_rpc_url")
	srcClient = libs.GetClientByUrl(srcRpcUrl)
	dstRpcUrl := os.Getenv("dst_rpc_url")
	dstClient = libs.GetClientByUrl(dstRpcUrl)
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
		db.Exec("create table block (id bigint,hash text,info text)")
	}
	err = db.QueryRow("select id from block order by id desc limit 1").Scan(&lastSyncNum)
	if err != nil {
		log.Println(err)
	}
}
