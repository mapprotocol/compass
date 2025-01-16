package butter

type SolCrossInResp struct {
	Errno   int    `json:"errno"`
	Message string `json:"message"`
	Data    []struct {
		Route struct {
			Diff      string `json:"diff"`
			BridgeFee struct {
				Amount string `json:"amount"`
			} `json:"bridgeFee"`
			TradeType int `json:"tradeType"`
			GasFee    struct {
				Amount string `json:"amount"`
				Symbol string `json:"symbol"`
			} `json:"gasFee"`
			SwapFee struct {
				NativeFee string `json:"nativeFee"`
				TokenFee  string `json:"tokenFee"`
			} `json:"swapFee"`
			FeeConfig struct {
				FeeType         int    `json:"feeType"`
				Referrer        string `json:"referrer"`
				RateOrNativeFee int    `json:"rateOrNativeFee"`
			} `json:"feeConfig"`
			GasEstimated       string `json:"gasEstimated"`
			GasEstimatedTarget string `json:"gasEstimatedTarget"`
			TimeEstimated      int    `json:"timeEstimated"`
			Hash               string `json:"hash"`
			Timestamp          int64  `json:"timestamp"`
			HasLiquidity       bool   `json:"hasLiquidity"`
			SrcChain           struct {
				ChainID string `json:"chainId"`
				TokenIn struct {
					Address  string `json:"address"`
					Name     string `json:"name"`
					Decimals int    `json:"decimals"`
					Symbol   string `json:"symbol"`
					Icon     string `json:"icon"`
				} `json:"tokenIn"`
				TokenOut struct {
					Address  string `json:"address"`
					Name     string `json:"name"`
					Decimals int    `json:"decimals"`
					Symbol   string `json:"symbol"`
					Icon     string `json:"icon"`
				} `json:"tokenOut"`
				TotalAmountIn  string `json:"totalAmountIn"`
				TotalAmountOut string `json:"totalAmountOut"`
				Route          []struct {
					AmountIn  string        `json:"amountIn"`
					AmountOut string        `json:"amountOut"`
					DexName   string        `json:"dexName"`
					Path      []interface{} `json:"path"`
				} `json:"route"`
				Bridge string `json:"bridge"`
			} `json:"srcChain"`
			MinAmountOut struct {
				Amount string `json:"amount"`
				Symbol string `json:"symbol"`
			} `json:"minAmountOut"`
		} `json:"route"`
		TxParam []struct {
			To      string `json:"to"`
			ChainID string `json:"chainId"`
			Data    string `json:"data"`
			Value   string `json:"value"`
			Method  string `json:"method"`
		} `json:"txParam"`
		Error struct {
			Response struct {
				Errno   int    `json:"errno"`
				Message string `json:"message"`
			} `json:"response"`
			Status  int    `json:"status"`
			Message string `json:"message"`
			Name    string `json:"name"`
		} `json:"error"`
	} `json:"data"`
}
