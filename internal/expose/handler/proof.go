package handler

import (
	"crypto/ecdsa"
	"github.com/gin-gonic/gin"
	"github.com/mapprotocol/compass/internal/expose"
	"github.com/mapprotocol/compass/internal/expose/service"
	"github.com/mapprotocol/compass/internal/stream"
	"net/http"
)

type Expose struct {
	cfg      *expose.Config
	proofSrv *service.ProofSrv
}

func New(cfg *expose.Config, pri *ecdsa.PrivateKey) *Expose {
	return &Expose{proofSrv: service.NewProof(cfg, pri), cfg: cfg}
}

func (e *Expose) TxExec(c *gin.Context) {
	var req stream.TxExecOfRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	ret, err := e.proofSrv.TxExec(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	c.JSON(http.StatusOK, Success(map[string]interface{}{
		"data": ret,
	}))
}

func Error2Response(err error) interface{} {
	return map[string]interface{}{
		"code": 500,
		"msg":  err.Error(),
	}
}

func Success(data interface{}) interface{} {
	return map[string]interface{}{
		"code": 0,
		"msg":  "success",
		"data": data,
	}
}
