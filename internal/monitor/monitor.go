package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/internal/bsc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/internal/matic"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	utils "github.com/mapprotocol/compass/shared/ethereum"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Monitor struct {
	*chain.CommonSync
}

func New(cs *chain.CommonSync) *Monitor {
	return &Monitor{
		CommonSync: cs,
	}
}

func (m *Monitor) Sync() error {
	go func() {
		m.sync()
	}()
	return nil
}

// sync function of Monitor will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Monitor) sync() {
	for {
		time.Sleep(time.Hour)
	}
}

type Req struct {
	ChainId int64  `json:"chain_id"`
	Tx      string `json:"tx"`
}

func Handler(resp http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("Server Internal Error"))
		return
	}

	r := Req{}
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("Server Internal Error"))
		return
	}
	fmt.Println("rrrrrrr -------------- ", r, "string", string(bytes))

	cfg, ok := mapprotocol.OnlineChainCfg[msg.ChainId(r.ChainId)]
	if !ok {
		log.Info("Found a log that is not the current task ", "toChainID", r.ChainId)
		resp.WriteHeader(404)
		resp.Write([]byte(fmt.Sprintf("This ChainId(%d) Not Support", r.ChainId)))
		return
	}
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("Server Internal Error"))
		return
	}
	receipt, err := client.TransactionReceipt(context.Background(), common.HexToHash(r.Tx))
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("Server Internal Error"))
		return
	}
	find := false
	method := mapprotocol.MethodOfTransferIn
	log := receipt.Logs[0]
	for _, l := range receipt.Logs {
		if mtd, ok := mapprotocol.Event[l.Topics[0]]; ok {
			log = l
			method = mtd
			find = true
			break
		}
	}
	if !find {
		resp.WriteHeader(400)
		resp.Write([]byte("This Tx Not Match"))
		return
	}
	var data []byte
	switch strings.ToLower(mapprotocol.OnlineChaId[msg.ChainId(r.ChainId)]) {
	case chains.Bsc:
		data, err = bsc.GetProof(client, receipt.BlockNumber, log, method, msg.ChainId(r.ChainId))
	case chains.Map:
		data, err = utils.GetProof(client, receipt.BlockNumber, log, method, msg.ChainId(r.ChainId))
	case chains.Matic:
		matic.GetProof(client, receipt.BlockNumber, log, method, msg.ChainId(r.ChainId))
	case chains.Klaytn:
		kc, err := klaytn.DialHttp(cfg.Endpoint, true)
		if err != nil {
			resp.WriteHeader(500)
			resp.Write([]byte("Klaytn InitConn Failed, Server Internal Error"))
			return
		}
		data, err = klaytn.GetProof(client, kc, receipt.BlockNumber, log, method, msg.ChainId(r.ChainId))
	case chains.Eth2:
		data, err = eth2.GetProof(client, receipt.BlockNumber, log, method, msg.ChainId(r.ChainId))
	default:
	}
	client.Close()
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("Server Internal Error"))
		return
	}
	ret := map[string]interface{}{
		"proof": "0x" + common.Bytes2Hex(data),
	}

	d, _ := json.Marshal(ret)
	_, _ = resp.Write(d)
}
