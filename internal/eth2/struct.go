package eth2

type LightClientUpdate struct {
	AttestedBeaconHeader AttestedHeader        `json:"attested_beacon_header"`
	SyncAggregate        SyncAggregate         `json:"sync_aggregate"`
	SignatureSlot        Slot                  `json:"signature_slot"`  // todo 难点
	FinalizedUpdate      FinalizedHeaderUpdate `json:"finality_update"` // todo 难点
	SyncCommitteeUpdate  SyncCommitteeUpdate   `json:"sync_committee_update"`
}

type Slot uint64

type FinalizedHeaderUpdate struct {
	HeaderUpdate   HeaderUpdate `json:"header_update"`
	FinalityBranch []string     `json:"finality_branch"`
}

type HeaderUpdate struct {
	BeaconHeader        AttestedHeader `json:"beacon_header"`
	ExecutionBlockHash  string         `json:"execution_block_hash"`
	ExecutionHashBranch []string       `json:"execution_hash_branch"`
}

type SyncCommitteeUpdate struct {
	NextSyncCommittee       NextSyncCommittee `json:"next_sync_committee"`
	NextSyncCommitteeBranch []string          `json:"next_sync_committee_branch"`
}
