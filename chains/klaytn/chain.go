package klaytn

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/tx"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/klaytn/klaytn/common"

	"github.com/ChainSafe/log15"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var (
	kClient = &klaytn.Client{}
)

func connectKClient(endpoint string) error {
	kc, err := klaytn.DialHttp(endpoint, true)
	if err != nil {
		return err
	}
	kClient = kc
	return nil
}

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	err := connectKClient(chainCfg.Endpoint)
	if err != nil {
		return nil, err
	}

	return chain.New(chainCfg, logger, sysErr, role, connection.NewConnection,
		chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfAssembleProof(assembleProof),
		chain.OptOfOracleHandler(chain.DefaultOracleHandler))
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	if err := syncValidatorHeader(m, latestBlock); err != nil {
		return err
	}

	if err := syncHeader(m, latestBlock); err != nil {
		return err
	}

	return nil
}

func syncValidatorHeader(m *chain.Maintainer, latestBlock *big.Int) error {
	kHeader, err := kClient.BlockByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}

	if kHeader.VoteData == "0x" {
		return nil
	}
	m.Log.Info("Get voteData", "blockHeight", latestBlock, "voteData", kHeader.VoteData)
	data := common.Hex2Bytes(strings.TrimPrefix(kHeader.VoteData, klaytn.PrefixOfHex))
	gVote := new(klaytn.GovernanceVote)
	err = rlp.DecodeBytes(data, gVote)
	if err != nil {
		m.Log.Error("Failed to decode a vote", "number", kHeader.Number, "key", gVote.Key, "value", gVote.Value, "validator", gVote.Validator)
		return err
	}

	if gVote.Key != "governance.addvalidator" && gVote.Key != "governance.removevalidator" {
		return nil
	}

	time.Sleep(time.Second)
	m.Log.Info("Send Validator Header", "blockHeight", latestBlock)
	return sendSyncHeader(m, latestBlock, 2)
}

func syncHeader(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfKlaytn))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}

	m.Log.Info("Find sync block", "current height", latestBlock)
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

	return sendSyncHeader(m, latestBlock, mapprotocol.HeaderOneCount)
}

func sendSyncHeader(m *chain.Maintainer, latestBlock *big.Int, count int) error {
	headers, err := assembleHeader(m.Conn.Client(), latestBlock, count)
	if err != nil {
		return err
	}

	input, err := mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(headers)
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

func assembleHeader(client *ethclient.Client, latestBlock *big.Int, count int) ([]klaytn.Header, error) {
	headers := make([]klaytn.Header, count)
	for i := 0; i < count; i++ {
		headerHeight := new(big.Int).Add(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := client.HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, err
		}
		hKheader, err := kClient.BlockByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, err
		}

		headers[count-count+i] = klaytn.ConvertContractHeader(header, hKheader)
	}

	return headers, nil
}

func assembleProof(m *chain.Messenger, log *types.Log, proofType int64, toChainID uint64, sign [][]byte) (*msg.Message, error) {
	var (
		message   msg.Message
		orderId   = log.Topics[1]
		method    = m.GetMethod(log.Topics[0])
		bigNumber = big.NewInt(int64(log.BlockNumber))
	)

	txsHash, err := klaytn.GetTxsHashByBlockNumber(kClient, bigNumber)
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
	if err != nil {
		return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}
	// get block
	header, err := m.Conn.Client().HeaderByNumber(context.Background(), bigNumber)
	if err != nil {
		return nil, err
	}
	kHeader, err := kClient.BlockByNumber(context.Background(), bigNumber)
	if err != nil {
		return nil, err
	}

	payload, err := klaytn.AssembleProof(kClient, klaytn.ConvertContractHeader(header, kHeader), log, m.Cfg.Id, receipts, method, proofType)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	msgPayload := []interface{}{payload, orderId, log.BlockNumber, log.TxHash}
	message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
	return &message, nil
}
