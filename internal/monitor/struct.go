package monitor

import "time"

type BridgeTransactionInfo struct {
	Id            int         `gorm:"column:id" db:"id" json:"id" form:"id"`
	SourceChainId interface{} `gorm:"column:source_chain_id" db:"source_chain_id" json:"source_chain_id" form:"source_chain_id"`
	SourceHash    interface{} `gorm:"column:source_hash" db:"source_hash" json:"source_hash" form:"source_hash"`
	CompleteTime  *time.Time  `gorm:"column:complete_time" db:"complete_time" json:"complete_time" form:"complete_time"`
	CreatedAt     *time.Time  `gorm:"column:created_at" db:"created_at" json:"created_at" form:"created_at"`
}
