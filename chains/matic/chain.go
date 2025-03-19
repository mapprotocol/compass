package matic

import (
	"context"
	"fmt"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/matic"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"sync"
)

type Chain struct {
}

func New() *Chain {
	return &Chain{}
}

func (c *Chain) New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, role, connection.NewConnection,
		chain.OptOfSync2Map(c.syncHeaderToMap),
		chain.OptOfInitHeight(12),
		chain.OptOfOracleHandler(chain.DefaultOracleHandler),
		chain.OptOfAssembleProof(c.assembleProof),
	)
}

func (c *Chain) syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, mapprotocol.ConfirmsOfMatic), big.NewInt(mapprotocol.HeaderCountOfMatic))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	if err != nil {
		m.Log.Error("Get current synced Height failed", "err", err)
		return err
	}
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.Log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}

	m.Log.Info("Find sync block", "current height", latestBlock)
	startBlock := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.ConfirmsOfMatic.Int64()+1))
	headers := make([]*types.Header, mapprotocol.ConfirmsOfMatic.Int64())
	for i := 0; i < int(mapprotocol.ConfirmsOfMatic.Int64()); i++ {
		headerHeight := new(big.Int).Add(startBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		headers[i] = header
	}

	mHeaders := make([]matic.BlockHeader, 0, len(headers))
	for _, h := range headers {
		mHeaders = append(mHeaders, matic.ConvertHeader(h))
	}

	input, err := mapprotocol.Matic.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(mHeaders)
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	msgpayload := []interface{}{id, input}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return err
	}

	err = m.WaitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chain) assembleProof(m *chain.Messenger, log *types.Log, proofType int64, toChainID uint64, sign [][]byte) (*msg.Message, error) {
	var (
		message msg.Message
		orderId = log.Topics[1]
	)
	var orderId32 [32]byte
	for idx, v := range orderId {
		orderId32[idx] = v
	}
	payload, err := c.Proof(m.Conn.Client(), log, "", proofType, uint64(m.Cfg.Id), toChainID, sign)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	msgPayload := []interface{}{payload, orderId32, log.BlockNumber, log.TxHash}
	message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
	return &message, nil
}

func (c *Chain) Connect(id, endpoint, mcs, lightNode, oracleNode string) (*ethclient.Client, error) {
	conn := connection.NewConnection(endpoint, true, nil, nil, big.NewInt(chain.DefaultGasLimit),
		big.NewInt(chain.DefaultGasPrice), chain.DefaultGasMultiplier)
	err := conn.Connect()
	if err != nil {
		return nil, err
	}

	fn := sync.OnceFunc(func() {
		idInt, _ := strconv.ParseUint(id, 10, 64)
		oracleAbi, _ := abi.New(mapprotocol.OracleAbiJson)
		call := contract.New(conn, []common.Address{common.HexToAddress(mcs)}, oracleAbi)
		mapprotocol.ContractMapping[msg.ChainId(idInt)] = call

		oAbi, _ := abi.New(mapprotocol.SignerJson)
		oracleCall := contract.New(conn, []common.Address{common.HexToAddress(oracleNode)}, oAbi)
		mapprotocol.SingMapping[msg.ChainId(idInt)] = oracleCall

		fn := mapprotocol.Map2EthHeight(constant.ZeroAddress.Hex(), common.HexToAddress(lightNode), conn.Client())
		mapprotocol.Map2OtherHeight[msg.ChainId(idInt)] = fn
	})
	fn()

	return conn.Client(), nil
}

func (c *Chain) Proof(client *ethclient.Client, log *types.Log, endpoint string, proofType int64, selfId,
	toChainID uint64, sign [][]byte) ([]byte, error) {
	var (
		orderId   = log.Topics[1]
		method    = chain.GetMethod(log.Topics[0])
		bigNumber = big.NewInt(int64(log.BlockNumber))
	)
	txsHash, err := tx.GetTxsHashByBlockNumber(client, bigNumber)
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	var receipts []*types.Receipt
	key := strconv.FormatUint(selfId, 10) + "_" + bigNumber.String()
	if v, ok := proof.CacheReceipt[key]; ok {
		receipts = v
	} else {
		tmp, err := tx.GetMaticReceiptsByTxsHash(client, txsHash)
		if err != nil {
			return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}
		for _, t := range tmp {
			if t == nil {
				continue
			}
			receipts = append(receipts, t)
		}
		proof.CacheReceipt[key] = receipts
	}

	var orderId32 [32]byte
	for idx, v := range orderId {
		orderId32[idx] = v
	}
	headers := make([]*types.Header, mapprotocol.ConfirmsOfMatic.Int64())
	for i := 0; i < int(mapprotocol.ConfirmsOfMatic.Int64()); i++ {
		headerHeight := new(big.Int).Add(bigNumber, new(big.Int).SetInt64(int64(i)))
		tmp, err := client.HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, fmt.Errorf("getHeader failed, err is %v", err)
		}
		headers[i] = tmp
	}

	mHeaders := make([]matic.BlockHeader, 0, len(headers))
	for _, h := range headers {
		mHeaders = append(mHeaders, matic.ConvertHeader(h))
	}

	payload, err := matic.AssembleProof(mHeaders, log, msg.ChainId(selfId), receipts, method, proofType, orderId32)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	return payload, nil
}

func (c *Chain) Maintainer(client *ethclient.Client, selfId, toChainId uint64, srcEndpoint string) ([]byte, error) {
	syncedHeight, err := mapprotocol.Get2MapHeight(msg.ChainId(selfId))
	if err != nil {
		return nil, errors.Wrap(err, "unable to get synced height")
	}
	startBlock := big.NewInt(syncedHeight.Int64() + mapprotocol.HeaderCountOfMatic)
	headers := make([]*types.Header, mapprotocol.ConfirmsOfMatic.Int64())
	for i := 0; i < int(mapprotocol.ConfirmsOfMatic.Int64()); i++ {
		headerHeight := new(big.Int).Add(startBlock, new(big.Int).SetInt64(int64(i)))
		header, err := client.HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, err
		}
		headers[i] = header
	}

	mHeaders := make([]matic.BlockHeader, 0, len(headers))
	for _, h := range headers {
		mHeaders = append(mHeaders, matic.ConvertHeader(h))
	}

	input, err := mapprotocol.Matic.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(mHeaders)
	if err != nil {
		return nil, err
	}

	ret, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodUpdateBlockHeader, big.NewInt(0).SetUint64(selfId), input)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
