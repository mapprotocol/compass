package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/mapprotocol/compass/pkg/etcd"

	"github.com/google/uuid"
	"github.com/mapprotocol/compass/msg"
)

const (
	PrefixOfCompass = "/compass/node/%s/%s"
)

var key string

type Compass struct {
	Ip          string                 `json:"ip"`
	Hostname    string                 `json:"hostname"`
	OnlineChain map[msg.ChainId]string `json:"online_chain"`
}

func Register(role string, oc map[msg.ChainId]string) error {
	uid, err := uUID()
	if err != nil {
		return err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	ip, err := localIP()
	if err != nil {
		return err
	}
	c := Compass{
		Ip:          ip.String(),
		Hostname:    hostname,
		OnlineChain: oc,
	}
	data, _ := json.Marshal(&c)
	key = fmt.Sprintf(PrefixOfCompass, role, uid)
	err = etcd.Put(context.Background(), key, string(data))
	if err != nil {
		return err
	}
	return nil
}

func UnRegister() error {
	err := etcd.Delete(context.Background(), key)
	if err != nil {
		return err
	}
	return nil
}

func localIP() (net.IP, error) {
	tables, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, t := range tables {
		addrs, err := t.Addrs()
		if err != nil {
			return nil, err
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			if v4 := ipnet.IP.To4(); v4 != nil {
				return v4, nil
			}
		}
	}
	return nil, fmt.Errorf("cannot find local IP address")
}

func uUID() (string, error) {
	u, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
