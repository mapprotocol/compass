package eth2

import "github.com/mapprotocol/compass/internal/constant"

func GetPeriodForSlot(slot int64) int64 {
	return slot / (constant.SlotsPerEpoch * constant.EpochsPerPeriod)
}
