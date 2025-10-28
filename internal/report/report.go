package report

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/client"
)

var (
	UrlOfReport = "/api/common/verify/hash"
)

var (
	defaultReport *Report
)

func Init(domain string) {
	defaultReport = New(domain)
}

func Add(data *Data) {
	defaultReport.Add(data)
	defaultReport.Start()
}

type Reportable interface {
	Report()
}

type Data struct {
	OrderId string
	Hash    string
	IsRelay bool
}

type Report struct {
	log    log.Logger
	domain string
	ch     chan *Data
}

func New(domain string) *Report {
	ll := log.Root().New("func", "reporter")
	return &Report{
		log:    ll,
		ch:     make(chan *Data, 100),
		domain: domain,
	}
}

func (r *Report) Add(data *Data) {
	r.ch <- data
}

func (r *Report) Report(data *Data) {
	requestMap := map[string]interface{}{
		"orderId": data.OrderId,
		"hash":    data.Hash,
		"type":    r.convertType(data.IsRelay),
	}
	reqData, _ := json.Marshal(&requestMap)
	resp, err := client.JsonPost(fmt.Sprintf("%s%s", r.domain, UrlOfReport), reqData)
	if err != nil {
		log.Error("report error", "err", err)
		return
	}
	log.Info("report success", "response", string(resp))
}

func (r *Report) convertType(isRelay bool) string {
	switch isRelay {
	case true:
		return "relay"
	default:
		return "dest"
	}
}

func (r *Report) Start() {
	r.log.Info("Reporter started")
	go func() {
		select {
		case data, ok := <-r.ch:
			if !ok {
				return
			}
			r.Report(data)
		default:
			time.Sleep(time.Millisecond * 500)
		}
		r.log.Info("Reporter stopped")
	}()
}

func (r *Report) Stop() {
	close(r.ch)
}
