package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	log "github.com/ChainSafe/log15"
)

func TestSwapFailedMinTxAgeIsSixMinutes(t *testing.T) {
	if minTxAge != 6*time.Minute {
		t.Fatalf("minTxAge = %s, want 6m", minTxAge)
	}
}

func TestRescueSourceHash(t *testing.T) {
	for _, tc := range []struct {
		name string
		tx   pendingTx
		want string
	}{
		{"eth to map failed", pendingTx{State: stateRelayFailed, SourceHash: "eth", RelayHash: "failed-map", RelayInHash: "map-in"}, "eth"},
		{"eth to map retry", pendingTx{State: stateRelayRetry, SourceHash: "eth", RelayInHash: "map-in"}, "eth"},
		{"map to bsc pending", pendingTx{State: stateRelayConfirmed, SourceHash: "eth", RelayHash: "map"}, "map"},
		{"map to bsc failed", pendingTx{State: stateDestFailed, SourceHash: "eth", RelayHash: "map", RelayInHash: "map-in"}, "map"},
		{"destination swap failed", pendingTx{State: stateDestSwapFailed, SourceHash: "eth", RelayHash: "map"}, "map"},
		{"relay hash fallback", pendingTx{State: stateDestFailed, SourceHash: "eth", RelayInHash: "map-in"}, "map-in"},
		{"direct source fallback", pendingTx{State: stateDestFailed, SourceHash: "source"}, "source"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := rescueSourceHash(tc.tx); got != tc.want {
				t.Fatalf("rescueSourceHash() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatAffiliatesForLog(t *testing.T) {
	tests := []struct {
		name       string
		affiliates []affiliate
		want       string
	}{
		{
			name: "multiple affiliates",
			affiliates: []affiliate{
				{ID: 6, Name: "tokenpocket"},
				{ID: 28, Name: "TokenPocket"},
			},
			want: `[{"id":6,"name":"tokenpocket"},{"id":28,"name":"TokenPocket"}]`,
		},
		{name: "empty affiliates", want: `[]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAffiliatesForLog(tt.affiliates); got != tt.want {
				t.Fatalf("formatAffiliatesForLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecordAttemptIncludesAllAffiliatesInRetryLogs(t *testing.T) {
	id := t.Name()
	seenFailedMu.Lock()
	delete(seenFailed, id)
	seenFailedMu.Unlock()
	t.Cleanup(func() {
		seenFailedMu.Lock()
		delete(seenFailed, id)
		seenFailedMu.Unlock()
	})

	var records []log.Record
	logger := log.New()
	logger.SetHandler(log.FuncHandler(func(record *log.Record) error {
		copied := *record
		copied.Ctx = append([]interface{}(nil), record.Ctx...)
		records = append(records, copied)
		return nil
	}))

	affiliates := []affiliate{
		{ID: 6, Name: "tokenpocket"},
		{ID: 28, Name: "TokenPocket"},
	}
	recordAttempt(id, "order-1", affiliates, errors.New("send failed"), logger)

	if len(records) != 2 {
		t.Fatalf("recordAttempt emitted %d records, want 2", len(records))
	}
	want := `[{"id":6,"name":"tokenpocket"},{"id":28,"name":"TokenPocket"}]`
	for _, record := range records {
		var got interface{}
		for i := 0; i+1 < len(record.Ctx); i += 2 {
			if record.Ctx[i] == "affiliates" {
				got = record.Ctx[i+1]
				break
			}
		}
		if got != want {
			t.Fatalf("%q affiliates = %#v, want %q", record.Msg, got, want)
		}
	}
}

func TestIsTokenProjectTransactionCaseInsensitive(t *testing.T) {
	for _, name := range []string{"tokenProject", "TokenProject", "TOKENPROJECT", "tp", "TP"} {
		t.Run(name, func(t *testing.T) {
			tx := pendingTx{Affiliates: []affiliate{{Name: name}}}
			if !isTokenProjectTransaction(tx) {
				t.Fatalf("isTokenProjectTransaction returned false for %q", name)
			}
		})
	}

	for _, tx := range []pendingTx{
		{},
		{Affiliates: []affiliate{{Name: "butter"}}},
	} {
		if isTokenProjectTransaction(tx) {
			t.Fatalf("isTokenProjectTransaction returned true for %+v", tx.Affiliates)
		}
	}
}

func TestPickTxParamsForTokenProjectUsesFirstNormalRoute(t *testing.T) {
	data := &execData{
		UserRouter: true,
		ExecRoute: &execRoute{
			RescueFundsTxParam: &txParam{Method: "refund"},
			RouteWithTxParams: []routeWithTx{
				{TxParam: []txParam{{Method: "approve"}, {Method: "bridge"}}},
				{TxParam: []txParam{{Method: "other-route"}}},
			},
		},
	}

	got, err := pickTxParams(data, true)
	if err != nil {
		t.Fatalf("pickTxParams returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("pickTxParams returned %d params, want 2", len(got))
	}
	if got[0].Method != "approve" || got[1].Method != "bridge" {
		t.Fatalf("pickTxParams returned methods %q, %q", got[0].Method, got[1].Method)
	}
}

func TestPickTxParamsForTransactionRoutesTokenProjectAwayFromRefund(t *testing.T) {
	tx := pendingTx{Affiliates: []affiliate{{Name: "TP"}}}
	data := &execData{
		UserRouter: true,
		ExecRoute: &execRoute{
			RescueFundsTxParam: &txParam{Method: "refund"},
			RouteWithTxParams: []routeWithTx{
				{TxParam: []txParam{{Method: "bridge"}}},
			},
		},
	}

	got, err := pickTxParamsForTransaction(tx, data)
	if err != nil {
		t.Fatalf("pickTxParamsForTransaction returned error: %v", err)
	}
	if len(got) != 1 || got[0].Method != "bridge" {
		t.Fatalf("pickTxParamsForTransaction returned %+v, want normal bridge param", got)
	}
}

func TestPickTxParamsForTransactionRoutesTokenPocketAffiliateIDsAwayFromRefund(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{name: "6", payload: `{"affiliates":[{"id":6}]}`},
		{name: "28", payload: `{"affiliates":[{"id":28}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tx pendingTx
			if err := json.Unmarshal([]byte(tc.payload), &tx); err != nil {
				t.Fatalf("unmarshal pending transaction: %v", err)
			}
			data := &execData{
				UserRouter: true,
				ExecRoute: &execRoute{
					RescueFundsTxParam: &txParam{Method: "refund"},
					RouteWithTxParams: []routeWithTx{
						{TxParam: []txParam{{Method: "bridge"}}},
					},
				},
			}

			got, err := pickTxParamsForTransaction(tx, data)
			if err != nil {
				t.Fatalf("pickTxParamsForTransaction returned error: %v", err)
			}
			if len(got) != 1 || got[0].Method != "bridge" {
				t.Fatalf("pickTxParamsForTransaction returned %+v, want normal bridge param", got)
			}
		})
	}
}

func TestPickTxParamsForTokenProjectRejectsMissingNormalRoute(t *testing.T) {
	data := &execData{
		UserRouter: true,
		ExecRoute: &execRoute{
			RescueFundsTxParam: &txParam{Method: "refund"},
		},
	}

	_, err := pickTxParams(data, true)
	if err == nil {
		t.Fatal("pickTxParams returned nil error without a normal route")
	}
	if !strings.Contains(err.Error(), "routeWithTxParams") {
		t.Fatalf("error lacks normal route context: %v", err)
	}
}

func TestPickTxParamsForTokenProjectRejectsEmptyFirstRoute(t *testing.T) {
	data := &execData{
		UserRouter: true,
		ExecRoute: &execRoute{
			RescueFundsTxParam: &txParam{Method: "refund"},
			RouteWithTxParams:  []routeWithTx{{}},
		},
	}

	_, err := pickTxParams(data, true)
	if err == nil {
		t.Fatal("pickTxParams returned nil error for an empty first normal route")
	}
	if !strings.Contains(err.Error(), "routeWithTxParams[0].txParam") {
		t.Fatalf("error lacks empty normal route context: %v", err)
	}
}

func TestPickTxParamsForRegularTransactionKeepsRefund(t *testing.T) {
	tx := pendingTx{Affiliates: []affiliate{{ID: 7, Name: "butter"}}}
	data := &execData{
		UserRouter: true,
		ExecRoute: &execRoute{
			RescueFundsTxParam: &txParam{Method: "refund"},
			RouteWithTxParams: []routeWithTx{
				{TxParam: []txParam{{Method: "bridge"}}},
			},
		},
	}

	got, err := pickTxParamsForTransaction(tx, data)
	if err != nil {
		t.Fatalf("pickTxParams returned error: %v", err)
	}
	if len(got) != 1 || got[0].Method != "refund" {
		t.Fatalf("pickTxParams returned %+v, want refund param", got)
	}
}

func TestSendTxParamsSendsAllInOrder(t *testing.T) {
	params := []txParam{{Method: "approve"}, {Method: "bridge"}}
	var sent []string

	hashes, err := sendTxParams(params, func(param txParam) (string, error) {
		sent = append(sent, param.Method)
		return "hash-" + param.Method, nil
	})
	if err != nil {
		t.Fatalf("sendTxParams returned error: %v", err)
	}
	if strings.Join(sent, ",") != "approve,bridge" {
		t.Fatalf("send order was %v", sent)
	}
	if strings.Join(hashes, ",") != "hash-approve,hash-bridge" {
		t.Fatalf("hashes were %v", hashes)
	}
}

func TestSendTxParamsStopsAtFirstFailure(t *testing.T) {
	params := []txParam{{Method: "approve"}, {Method: "bridge"}, {Method: "must-not-send"}}
	wantErr := errors.New("bridge failed")
	var sent []string

	_, err := sendTxParams(params, func(param txParam) (string, error) {
		sent = append(sent, param.Method)
		if param.Method == "bridge" {
			return "", wantErr
		}
		return "hash-" + param.Method, nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("sendTxParams error = %v, want wrapped %v", err, wantErr)
	}
	if strings.Join(sent, ",") != "approve,bridge" {
		t.Fatalf("send order was %v; later params must not be sent", sent)
	}
}
