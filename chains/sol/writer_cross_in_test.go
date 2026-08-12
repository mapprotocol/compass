package sol

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/mapprotocol/compass/internal/butter"
)

func TestSendSolCrossInTxsContinuesExecuteWhenOpenAlreadyInUse(t *testing.T) {
	resp := mustSolCrossInResp(t, `{
		"data": [{
			"txParam": [
				{"step": "receiveOpen", "data": "open-data"},
				{"step": "receiveExecute", "data": "execute-data"}
			]
		}]
	}`)
	var calls []string
	send := func(txParam butter.SolCrossInTxParam, _ int) (string, error) {
		calls = append(calls, txParam.Step)
		if txParam.Step == "receiveOpen" {
			return "", errors.New("Allocate: account Already In Use")
		}
		return "execute-hash", nil
	}

	txHashes, openDone, err := sendSolCrossInTxs(resp, false, send)
	if err != nil {
		t.Fatalf("sendSolCrossInTxs() error = %v", err)
	}
	if !openDone {
		t.Fatal("sendSolCrossInTxs() openDone = false, want true")
	}
	if want := []string{"receiveOpen", "receiveExecute"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("send calls = %v, want %v", calls, want)
	}
	if want := []string{"execute-hash"}; !reflect.DeepEqual(txHashes, want) {
		t.Fatalf("tx hashes = %v, want %v", txHashes, want)
	}
}

func TestSendSolCrossInTxsReturnsOtherOpenErrors(t *testing.T) {
	resp := mustSolCrossInResp(t, `{
		"data": [{
			"txParam": [
				{"step": "receiveOpen", "data": "open-data"},
				{"step": "receiveExecute", "data": "execute-data"}
			]
		}]
	}`)
	errOpen := errors.New("REVERT opcode executed")
	var calls []string
	send := func(txParam butter.SolCrossInTxParam, _ int) (string, error) {
		calls = append(calls, txParam.Step)
		return "", errOpen
	}

	txHashes, openDone, err := sendSolCrossInTxs(resp, false, send)
	if !errors.Is(err, errOpen) {
		t.Fatalf("sendSolCrossInTxs() error = %v, want %v", err, errOpen)
	}
	if openDone {
		t.Fatal("sendSolCrossInTxs() openDone = true, want false")
	}
	if txHashes != nil {
		t.Fatalf("tx hashes = %v, want nil", txHashes)
	}
	if want := []string{"receiveOpen"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("send calls = %v, want %v", calls, want)
	}
}

func TestSendSolCrossInTxsKeepsOpenDoneWhenExecuteFailsAfterAlreadyInUse(t *testing.T) {
	resp := mustSolCrossInResp(t, `{
		"data": [{
			"txParam": [
				{"step": "receiveOpen", "data": "open-data"},
				{"step": "receiveExecute", "data": "execute-data"}
			]
		}]
	}`)
	errExecute := errors.New("execute failed")
	var calls []string
	send := func(txParam butter.SolCrossInTxParam, _ int) (string, error) {
		calls = append(calls, txParam.Step)
		if txParam.Step == "receiveOpen" {
			return "", errors.New("account already in use")
		}
		return "", errExecute
	}

	txHashes, openDone, err := sendSolCrossInTxs(resp, false, send)
	if !errors.Is(err, errExecute) {
		t.Fatalf("sendSolCrossInTxs() error = %v, want %v", err, errExecute)
	}
	if !openDone {
		t.Fatal("sendSolCrossInTxs() openDone = false, want true")
	}
	if len(txHashes) != 0 {
		t.Fatalf("tx hashes = %v, want empty", txHashes)
	}
	if want := []string{"receiveOpen", "receiveExecute"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("send calls = %v, want %v", calls, want)
	}
}

func mustSolCrossInResp(t *testing.T, body string) *butter.SolCrossInResp {
	t.Helper()
	var resp butter.SolCrossInResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal solCrossIn response: %v", err)
	}
	return &resp
}
