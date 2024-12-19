// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/contract"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"math/big"

	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/ethereum/go-ethereum"
	goeth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

type GetHeight func() (*big.Int, error)
type GetVerifyRange func() (*big.Int, *big.Int, error)

var (
	MapId                string
	GlobalMapConn        *ethclient.Client
	SyncOtherMap         = make(map[msg.ChainId]*big.Int)  // map to other chain init height
	Map2OtherHeight      = make(map[msg.ChainId]GetHeight) // get map to other height function collect
	ContractMapping      = make(map[msg.ChainId]*contract.Call)
	LightNodeMapping     = make(map[msg.ChainId]*contract.Call)
	SingMapping          = make(map[msg.ChainId]*contract.Call)
	MosMapping           = make(map[msg.ChainId]string)
	Get2MapHeight        = func(chainId msg.ChainId) (*big.Int, error) { return nil, nil }                // get other chain to map height
	GetEth22MapNumber    = func(chainId msg.ChainId) (*big.Int, *big.Int, error) { return nil, nil, nil } // can reform, return data is []byte
	GetDataByManager     = func(string, ...interface{}) ([]byte, error) { return nil, nil }
	GetNodeTypeByManager = func(string, ...interface{}) (*big.Int, error) { return nil, nil }
)

func InitLightManager(lightNode common.Address) {
	GetDataByManager = func(method string, params ...interface{}) ([]byte, error) {
		input, err := PackInput(LightManger, method, params...)
		if err != nil {
			return nil, errors.Wrap(err, "get other2map packInput failed")
		}
		output, err := GlobalMapConn.CallContract(
			context.Background(),
			goeth.CallMsg{From: constant.ZeroAddress, To: &lightNode, Data: input},
			nil,
		)
		if err != nil {
			return nil, err
		}
		outputs := LightManger.Methods[method].Outputs
		unpack, err := outputs.Unpack(output)
		if err != nil {
			return nil, err
		}
		ret := make([]byte, 0)
		if err = outputs.Copy(&ret, unpack); err != nil {
			return nil, err
		}

		return ret, nil
	}
}

func Init2GetEth22MapNumber(lightNode common.Address) {
	GetEth22MapNumber = func(chainId msg.ChainId) (*big.Int, *big.Int, error) {
		input, err := PackInput(LightManger, MethodClientState, big.NewInt(int64(chainId)))
		if err != nil {
			return nil, nil, errors.Wrap(err, "get eth22map packInput failed")
		}

		output, err := GlobalMapConn.CallContract(context.Background(),
			goeth.CallMsg{From: constant.ZeroAddress, To: &lightNode, Data: input}, nil)
		if err != nil {
			return nil, nil, err
		}

		outputs := LightManger.Methods[MethodClientState].Outputs
		unpack, err := outputs.Unpack(output)
		if err != nil {
			return nil, nil, err
		}

		back := make([]byte, 0)
		if err = outputs.Copy(&back, unpack); err != nil {
			return nil, nil, err
		}

		ret := struct {
			StartNumber *big.Int
			EndNumber   *big.Int
		}{}
		analysis, err := Eth2.Methods[MethodClientStateAnalysis].Outputs.Unpack(back)
		if err != nil {
			return nil, nil, errors.Wrap(err, "analysis")
		}
		if err = Eth2.Methods[MethodClientStateAnalysis].Outputs.Copy(&ret, analysis); err != nil {
			return nil, nil, errors.Wrap(err, "analysis copy")
		}

		return ret.StartNumber, ret.EndNumber, nil
	}
}

func InitOtherChain2MapHeight(lightManager common.Address) {
	Get2MapHeight = func(chainId msg.ChainId) (*big.Int, error) {
		input, err := PackInput(LightManger, MethodOfHeaderHeight, big.NewInt(int64(chainId)))
		if err != nil {
			return nil, errors.Wrap(err, "get other2map by manager packInput failed")
		}

		height, err := HeaderHeight(lightManager, input)
		if err != nil {
			return nil, errors.Wrap(err, "get other2map headerHeight by lightManager failed")
		}
		return height, nil
	}
}

func Map2EthHeight(fromUser string, lightNode common.Address, client *ethclient.Client) GetHeight {
	return func() (*big.Int, error) {
		from := common.HexToAddress(fromUser)
		input, err := PackInput(Height, MethodOfHeaderHeight)
		if err != nil {
			return nil, fmt.Errorf("pack lightNode headerHeight Input failed, err is %v", err.Error())
		}
		output, err := client.CallContract(context.Background(),
			ethereum.CallMsg{
				From: from,
				To:   &lightNode,
				Data: input,
			},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("headerHeight callContract failed, err is %v", err.Error())
		}

		return UnpackHeaderHeightOutput(output)
	}
}

func Map2NearHeight(lightNode string, client *nearclient.Client) GetHeight {
	return func() (*big.Int, error) {
		res, err := client.ContractViewCallFunction(context.Background(), lightNode, NearHeaderHeight,
			"e30=", block.FinalityFinal())
		if err != nil {
			return nil, errors.Wrap(err, "call near lightNode to headerHeight failed")
		}

		if res.Error != nil {
			return nil, fmt.Errorf("call near lightNode to get headerHeight resp exist error(%v)", *res.Error)
		}

		result := "" // use string return
		err = json.Unmarshal(res.Result, &result)
		if err != nil {
			return nil, errors.Wrap(err, "near lightNode headerHeight resp json marshal failed")
		}
		ret := new(big.Int)
		ret.SetString(result, 10)
		return ret, nil
	}
}

func PackInput(commonAbi abi.ABI, abiMethod string, params ...interface{}) ([]byte, error) {
	input, err := commonAbi.Pack(abiMethod, params...)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func UnpackHeaderHeightOutput(output []byte) (*big.Int, error) {
	outputs := Height.Methods[MethodOfHeaderHeight].Outputs
	unpack, err := outputs.Unpack(output)
	if err != nil {
		return big.NewInt(0), err
	}

	height := new(big.Int)
	if err = outputs.Copy(&height, unpack); err != nil {
		return big.NewInt(0), err
	}
	return height, nil
}

func HeaderHeight(to common.Address, input []byte) (*big.Int, error) {
	output, err := GlobalMapConn.CallContract(context.Background(), goeth.CallMsg{From: constant.ZeroAddress, To: &to, Data: input}, nil)
	if err != nil {
		return nil, err
	}
	height, err := UnpackHeaderHeightOutput(output)
	if err != nil {
		return nil, err
	}
	return height, nil
}

func LightManagerNodeType(lightNode common.Address) {
	GetNodeTypeByManager = func(method string, params ...interface{}) (*big.Int, error) {
		input, err := PackInput(LightManger, method, params...)
		if err != nil {
			return nil, errors.Wrap(err, "get other2map packInput failed")
		}
		output, err := GlobalMapConn.CallContract(
			context.Background(),
			goeth.CallMsg{From: constant.ZeroAddress, To: &lightNode, Data: input},
			nil,
		)
		if err != nil {
			return nil, err
		}
		outputs := LightManger.Methods[method].Outputs
		unpack, err := outputs.Unpack(output)
		if err != nil {
			return nil, err
		}
		ret := new(big.Int)
		if err = outputs.Copy(&ret, unpack); err != nil {
			return nil, err
		}

		return ret, nil
	}
}
