package stream

type BTCRawTransaction struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Txid     string `json:"txid"`
		Hash     string `json:"hash"`
		Version  int    `json:"version"`
		Size     int    `json:"size"`
		Vsize    int    `json:"vsize"`
		Weight   int    `json:"weight"`
		Locktime int    `json:"locktime"`
		Vin      []struct {
			Txid      string `json:"txid"`
			Vout      int    `json:"vout"`
			ScriptSig struct {
				Asm string `json:"asm"`
				Hex string `json:"hex"`
			} `json:"scriptSig"`
			Txinwitness []string `json:"txinwitness"`
			Sequence    int64    `json:"sequence"`
		} `json:"vin"`
		Vout []struct {
			Value        float64 `json:"value"`
			N            int     `json:"n"`
			ScriptPubKey struct {
				Asm     string `json:"asm"`
				Desc    string `json:"desc"`
				Hex     string `json:"hex"`
				Address string `json:"address"`
				Type    string `json:"type"`
			} `json:"scriptPubKey,omitempty"`
			ScriptPubKey0 struct {
				Asm  string `json:"asm"`
				Desc string `json:"desc"`
				Hex  string `json:"hex"`
				Type string `json:"type"`
			} `json:"scriptPubKey,omitempty"`
		} `json:"vout"`
		Hex           string `json:"hex"`
		Blockhash     string `json:"blockhash"`
		Confirmations int    `json:"confirmations"`
		Time          int    `json:"time"`
		Blocktime     int    `json:"blocktime"`
	} `json:"result"`
	ID    string `json:"id"`
	Error string `json:"error"`
}
