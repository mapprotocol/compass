package butter

type SolCrossInResp struct {
	Errno      int    `json:"errno"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       []struct {
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

type ExecSwapResp struct {
	Data struct {
		MinReceivedInLog   string `json:"minReceivedInLog"`
		RescueFundsTxParam struct {
			Args []struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"args"`
			ChainID string `json:"chainId"`
			Data    string `json:"data"`
			Method  string `json:"method"`
			To      string `json:"to"`
			Value   string `json:"value"`
		} `json:"rescueFundsTxParam"`
		RouteWithTxParams []struct {
			Route struct {
				BridgeFee struct {
					Amount string `json:"amount"`
				} `json:"bridgeFee"`
				Contract  string `json:"contract"`
				Diff      string `json:"diff"`
				Entrance  string `json:"entrance"`
				FeeConfig struct {
					FeeType         int    `json:"feeType"`
					RateOrNativeFee int    `json:"rateOrNativeFee"`
					Referrer        string `json:"referrer"`
				} `json:"feeConfig"`
				GasEstimated       string `json:"gasEstimated"`
				GasEstimatedTarget string `json:"gasEstimatedTarget"`
				GasFee             struct {
					Amount string `json:"amount"`
					InUSD  string `json:"inUSD"`
					Symbol string `json:"symbol"`
				} `json:"gasFee"`
				HasLiquidity bool   `json:"hasLiquidity"`
				Hash         string `json:"hash"`
				MinAmountOut struct {
					Amount string `json:"amount"`
					Symbol string `json:"symbol"`
				} `json:"minAmountOut"`
				SrcChain struct {
					Bridge  string `json:"bridge"`
					ChainID string `json:"chainId"`
					Route   []struct {
						AmountIn  string        `json:"amountIn"`
						AmountOut string        `json:"amountOut"`
						DexName   string        `json:"dexName"`
						Extra     interface{}   `json:"extra"`
						Path      []interface{} `json:"path"`
					} `json:"route"`
					TokenIn struct {
						Address  string `json:"address"`
						Decimals int    `json:"decimals"`
						Icon     string `json:"icon"`
						Name     string `json:"name"`
						Symbol   string `json:"symbol"`
					} `json:"tokenIn"`
					TokenOut struct {
						Address  string `json:"address"`
						Decimals int    `json:"decimals"`
						Icon     string `json:"icon"`
						Name     string `json:"name"`
						Symbol   string `json:"symbol"`
					} `json:"tokenOut"`
					TotalAmountIn     string `json:"totalAmountIn"`
					TotalAmountOut    string `json:"totalAmountOut"`
					TotalAmountOutUSD string `json:"totalAmountOutUSD"`
				} `json:"srcChain"`
				SwapFee struct {
					NativeFee string `json:"nativeFee"`
					TokenFee  string `json:"tokenFee"`
				} `json:"swapFee"`
				TimeEstimated     int    `json:"timeEstimated"`
				Timestamp         int64  `json:"timestamp"`
				TotalAmountInUSD  string `json:"totalAmountInUSD"`
				TotalAmountOutUSD string `json:"totalAmountOutUSD"`
				TradeType         int    `json:"tradeType"`
			} `json:"route"`
			TxParam []struct {
				Args []struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"args"`
				ChainID string `json:"chainId"`
				Data    string `json:"data"`
				Method  string `json:"method"`
				To      string `json:"to"`
				Value   string `json:"value"`
			} `json:"txParam"`
		} `json:"routeWithTxParams"`
	} `json:"data"`
	Errno   int    `json:"errno"`
	Message string `json:"message"`
}
