// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/mapprotocol/compass/core"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Connection struct {
	endpoint                  string
	http                      bool
	kp                        *keystore.Key
	gasLimit                  *big.Int
	maxGasPrice               *big.Int
	gasMultiplier             *big.Float
	conn                      *ethclient.Client
	opts                      *bind.TransactOpts
	callOpts                  *bind.CallOpts
	nonce                     uint64
	optsLock                  sync.Mutex
	log                       log15.Logger
	stop                      chan int // All routines should exit when this channel is closed
	reqTime, cacheBlockNumber int64
}

// NewConnection returns an uninitialized connection, must call Connection.Connect() before using.
func NewConnection(endpoint string, http bool, kp *keystore.Key, log log15.Logger, gasLimit, gasPrice *big.Int,
	gasMultiplier float64) core.Connection {
	bigFloat := new(big.Float).SetFloat64(gasMultiplier)
	return &Connection{
		endpoint:      endpoint,
		http:          http,
		kp:            kp,
		gasLimit:      gasLimit,
		maxGasPrice:   gasPrice,
		gasMultiplier: bigFloat,
		log:           log,
		stop:          make(chan int),
	}
}

// Connect starts the ethereum WS connection
func (c *Connection) Connect() error {
	var rpcClient *rpc.Client
	var err error
	// Start http or ws client
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	if c.http {
		rpcClient, err = rpc.DialHTTPWithClient(c.endpoint, client)
	} else {
		rpcClient, err = rpc.DialContext(context.Background(), c.endpoint)
	}
	if err != nil {
		return err
	}
	c.conn = ethclient.NewClient(rpcClient, c.endpoint, client)

	// Construct tx opts, call opts, and nonce mechanism
	opts, _, err := c.newTransactOpts(big.NewInt(0), c.gasLimit, c.maxGasPrice)
	if err != nil {
		return err
	}
	c.opts = opts
	c.nonce = 0
	return nil
}

// newTransactOpts builds the TransactOpts for the connection's keypair.
func (c *Connection) newTransactOpts(value, gasLimit, gasPrice *big.Int) (*bind.TransactOpts, uint64, error) {
	if c.kp == nil {
		return nil, 0, nil
	}
	privateKey := c.kp.PrivateKey
	address := ethcrypto.PubkeyToAddress(privateKey.PublicKey)

	nonce, err := c.conn.PendingNonceAt(context.Background(), address)
	if err != nil {
		return nil, 0, err
	}

	id, err := c.conn.ChainID(context.Background())
	if err != nil {
		return nil, 0, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, id)
	if err != nil {
		return nil, 0, err
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = value
	auth.GasLimit = uint64(gasLimit.Int64())
	auth.GasPrice = gasPrice
	auth.Context = context.Background()

	return auth, nonce, nil
}

func (c *Connection) Keypair() *keystore.Key {
	return c.kp
}

func (c *Connection) Client() *ethclient.Client {
	return c.conn
}

func (c *Connection) Opts() *bind.TransactOpts {
	return c.opts
}

func (c *Connection) CallOpts() *bind.CallOpts {
	return c.callOpts
}

func (c *Connection) SafeEstimateGas(ctx context.Context) (*big.Int, error) {
	var suggestedGasPrice *big.Int
	c.log.Debug("Fetching gasPrice from node")
	nodePriceEstimate, err := c.conn.SuggestGasPrice(context.TODO())
	if err != nil {
		return nil, err
	} else {
		suggestedGasPrice = nodePriceEstimate
	}

	gasPrice := multiplyGasPrice(suggestedGasPrice, c.gasMultiplier)

	// Check we aren't exceeding our limit
	if gasPrice.Cmp(c.maxGasPrice) == 1 {
		return c.maxGasPrice, nil
	} else {
		return gasPrice, nil
	}
}

func (c *Connection) EstimateGasLondon(ctx context.Context, baseFee *big.Int) (*big.Int, *big.Int, error) {
	var maxPriorityFeePerGas, maxFeePerGas *big.Int
	if c.maxGasPrice.Cmp(baseFee) < 0 {
		maxPriorityFeePerGas = big.NewInt(1000000)
		maxFeePerGas = new(big.Int).Add(c.maxGasPrice, maxPriorityFeePerGas)
		return maxPriorityFeePerGas, maxFeePerGas, nil
	}

	maxPriorityFeePerGas, err := c.conn.SuggestGasTipCap(context.TODO())
	if err != nil {
		return nil, nil, err
	}
	c.log.Info("EstimateGasLondon", "maxPriorityFeePerGas", maxPriorityFeePerGas)

	maxFeePerGas = new(big.Int).Add(
		maxPriorityFeePerGas,
		baseFee,
	)
	// mul
	maxFeePerGas = multiplyGasPrice(maxFeePerGas, c.gasMultiplier)

	// Check we aren't exceeding our limit
	if maxFeePerGas.Cmp(c.maxGasPrice) == 1 {
		c.log.Info("EstimateGasLondon maxFeePerGas more than set", "maxFeePerGas", maxFeePerGas, "baseFee", baseFee)
		maxPriorityFeePerGas.Sub(c.maxGasPrice, baseFee)
		maxFeePerGas = c.maxGasPrice
	}
	return maxPriorityFeePerGas, maxFeePerGas, nil
}

func multiplyGasPrice(gasEstimate *big.Int, gasMultiplier *big.Float) *big.Int {
	gasEstimateFloat := new(big.Float).SetInt(gasEstimate)
	result := gasEstimateFloat.Mul(gasEstimateFloat, gasMultiplier)
	gasPrice := new(big.Int)
	result.Int(gasPrice)
	return gasPrice
}

// LockAndUpdateOpts acquires a lock on the opts before updating the nonce
// and gas price.
func (c *Connection) LockAndUpdateOpts(needNewNonce bool) error {
	//c.optsLock.Lock()
	head, err := c.conn.HeaderByNumber(context.TODO(), nil)
	// cos map chain dont have this section in return,this err will be raised
	if err != nil && err.Error() != "missing required field 'sha3Uncles' for Header" {
		c.UnlockOpts()
		c.log.Error("LockAndUpdateOpts HeaderByNumber", "err", err)
		return err
	}

	if head.BaseFee != nil {
		c.opts.GasTipCap, c.opts.GasFeeCap, err = c.EstimateGasLondon(context.TODO(), head.BaseFee)
		// Both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) cannot be specified: https://github.com/ethereum/go-ethereum/blob/95bbd46eabc5d95d9fb2108ec232dd62df2f44ab/accounts/abi/bind/base.go#L254
		c.opts.GasPrice = nil
		if err != nil {
			// if EstimateGasLondon failed, fall back to suggestGasPrice
			c.opts.GasPrice, err = c.conn.SuggestGasPrice(context.TODO())
			if err != nil {
				//c.UnlockOpts()
				return err
			}
		}
		c.log.Info("LockAndUpdateOpts ", "head.BaseFee", head.BaseFee, "maxGasPrice", c.maxGasPrice,
			"gasTipCap", c.opts.GasTipCap, "gasFeeCap", c.opts.GasFeeCap)
	} else {
		var gasPrice *big.Int
		gasPrice, err = c.SafeEstimateGas(context.TODO())
		if err != nil {
			//c.UnlockOpts()
			return err
		}
		c.opts.GasPrice = gasPrice
	}

	if !needNewNonce {
		return nil
	}
	nonce, err := c.conn.PendingNonceAt(context.Background(), c.opts.From)
	if err != nil {
		//c.optsLock.Unlock()
		return err
	}
	c.opts.Nonce.SetUint64(nonce)
	return nil
}

func (c *Connection) UnlockOpts() {
	//c.optsLock.Unlock()
}

// LatestBlock returns the latest block from the current chain
func (c *Connection) LatestBlock() (*big.Int, error) {
	// 1s req
	if time.Now().Unix()-c.reqTime < constant.ReqInterval {
		return big.NewInt(0).SetInt64(c.cacheBlockNumber), nil
	}
	bnum, err := c.conn.BlockNumber(context.Background())
	if err != nil {
		return nil, err
	}
	c.cacheBlockNumber = int64(bnum)
	c.reqTime = time.Now().Unix()
	return big.NewInt(0).SetUint64(bnum), nil
}

// EnsureHasBytecode asserts if contract code exists at the specified address
func (c *Connection) EnsureHasBytecode(addr ethcommon.Address) error {
	//code, err := c.conn.CodeAt(context.Background(), addr, nil)
	//if err != nil {
	//	return err
	//}
	//
	//if len(code) == 0 {
	//	return fmt.Errorf("no bytecode found at %s", addr.Hex())
	//}
	return nil
}

// WaitForBlock will poll for the block number until the current block is equal or greater.
// If delay is provided it will wait until currBlock - delay = targetBlock
func (c *Connection) WaitForBlock(targetBlock *big.Int, delay *big.Int) error {
	for {
		select {
		case <-c.stop:
			return errors.New("connection terminated")
		default:
			currBlock, err := c.LatestBlock()
			if err != nil {
				return err
			}

			if delay != nil {
				currBlock.Sub(currBlock, delay)
			}

			// Equal or greater than target
			if currBlock.Cmp(targetBlock) >= 0 {
				return nil
			}
			c.log.Trace("Block not ready, waiting", "target", targetBlock, "current", currBlock, "delay", delay)
			time.Sleep(constant.BlockRetryInterval)
			continue
		}
	}
}

// Close terminates the client connection and stops any running routines
func (c *Connection) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	close(c.stop)
}

type DebugTransport struct {
	Base http.RoundTripper
}

func (dt *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	fmt.Println("=== 请求开始 ===")

	// 打印目标 URL
	fmt.Println("URL:", req.URL.String())

	// 调用底层 RoundTripper
	resp, err := dt.Base.RoundTrip(req)

	cost := time.Since(start)
	if err != nil {
		fmt.Println("请求失败:", err, "耗时:", cost)
		return nil, err
	}
	fmt.Println("收到响应头 耗时:", cost)
	return resp, nil
}
