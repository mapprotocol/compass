package sol

import (
	"reflect"
	"testing"
)

func TestDecodeRelayData(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    *MessageData
		wantErr bool
	}{
		{
			name: "relay",
			args: args{
				data: "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000005c1003030314220000000000000000000000000000000000000000000000986f0c7872706d6f730cdae5da23b64bfecd421d6487ffeabf6558828d7244764b4d4364596b67746e427759696568385634393936326a735948577369465500000000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeRelayData(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeRelayData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log("got ------ ", string(got.To))
			t.Log("got ------ ", got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecodeRelayData() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
