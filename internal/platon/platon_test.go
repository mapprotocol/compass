package platon

import "testing"

func Test_handlerValidatorSet(t *testing.T) {
	got, got1 := handlerValidatorSet("_xxxx")
	t.Log("got ------", got)
	t.Log("got1 ------", got1)
}
