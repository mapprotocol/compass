package butter

import (
	"fmt"
	"github.com/mapprotocol/compass/internal/client"
)

const (
	UrlOfExecSwap = "/execSwap"
)

var defaultButter = New()

type Butter struct {
}

func New() *Butter {
	return &Butter{}
}

func (b *Butter) ExecSwap(domain, query string) ([]byte, error) {
	return client.JsonGet(fmt.Sprintf("%s%s?%s", domain, UrlOfExecSwap, query))
}

func ExecSwap(domain, query string) ([]byte, error) {
	return defaultButter.ExecSwap(domain, query)
}
