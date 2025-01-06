package expose

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/internal/stream"
	"net/http"
)

type Expose struct {
	cfg *Config
}

func New(cfg *Config) *Expose {
	return &Expose{cfg: cfg}
}

func (e *Expose) FailedExec(c *gin.Context) {
	var req stream.FailedTxOfRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	data, err := butter.ExecSwap(e.cfg.Other.Butter, fmt.Sprintf("toChainId=%d&txHash=%s", req.ToChain, req.Hash))
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	ret := make(map[string]interface{})
	err = json.Unmarshal(data, &ret)
	if err != nil {
		c.JSON(http.StatusOK, Error2Response(err))
		return
	}

	c.JSON(http.StatusOK, ret)
}

func (e *Expose) SuccessProof(c *gin.Context) {
	var req stream.ProofOfRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}
	//
	//c.JSON(http.StatusOK, data)
}

func Error2Response(err error) interface{} {
	return map[string]interface{}{
		"code": 500,
		"msg":  err.Error(),
	}
}
