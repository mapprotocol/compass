package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSwapFailedMinTxAgeIsSixMinutes(t *testing.T) {
	if minTxAge != 6*time.Minute {
		t.Fatalf("minTxAge = %s, want 6m", minTxAge)
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
	data := &execData{
		UserRouter: true,
		ExecRoute: &execRoute{
			RescueFundsTxParam: &txParam{Method: "refund"},
			RouteWithTxParams: []routeWithTx{
				{TxParam: []txParam{{Method: "bridge"}}},
			},
		},
	}

	got, err := pickTxParams(data, false)
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
