package ton

import (
	"github.com/xssnick/tonutils-go/address"
	"testing"
)

func Test_convertToHex(t *testing.T) {
	type args struct {
		addr *address.Address
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "t-1",
			args: args{
				addr: address.MustParseAddr("EQAu3Kfn-NhsId75-WUCdpqPODvKzR0Fpc4MB7Szn0QOQiOW"),
			},
			want: "002edca7e7f8d86c21def9f96502769a8f383bcacd1d05a5ce0c07b4b39f440e42",
		},
		{
			name: "t-2",
			args: args{
				addr: address.MustParseAddr("kQAu3Kfn-NhsId75-WUCdpqPODvKzR0Fpc4MB7Szn0QOQpgc"),
			},
			want: "002edca7e7f8d86c21def9f96502769a8f383bcacd1d05a5ce0c07b4b39f440e42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertToHex(tt.args.addr); got != tt.want {
				t.Errorf("convertToHex() = %v, want %v", got, tt.want)
			}
		})
	}
}
