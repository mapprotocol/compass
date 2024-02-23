package mapprotocol

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

func TestGetZkProof(t *testing.T) {
	type args struct {
		endpoint string
		cid      msg.ChainId
		height   uint64
	}
	tests := []struct {
		name    string
		args    args
		want    []*big.Int
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				endpoint: "http://127.0.0.1:8181",
				cid:      212,
				height:   5000000,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetZkProof(tt.args.endpoint, tt.args.cid, tt.args.height)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetZkProof() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetZkProof() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCurValidators(t *testing.T) {
	rpcClient, _ := rpc.DialHTTP("https://testnet-rpc.maplabs.io")
	cli := ethclient.NewClient(rpcClient)
	type args struct {
		cli    *ethclient.Client
		number *big.Int
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				cli:    cli,
				number: big.NewInt(5000000),
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCurValidators(tt.args.cli, tt.args.number)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurValidators() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCurValidators() got = %v, want %v", got, tt.want)
			}
		})
	}
}
