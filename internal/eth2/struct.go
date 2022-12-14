package eth2

/*
pub struct LightClientUpdate {
    pub attested_beacon_header: BeaconBlockHeader,
    pub sync_aggregate: SyncAggregate,
    #[cfg_attr(
        not(target_arch = "wasm32"),
        serde(with = "eth2_serde_utils::quoted_u64")
    )]
    pub signature_slot: Slot,
    pub finality_update: FinalizedHeaderUpdate,
    pub sync_committee_update: Option<SyncCommitteeUpdate>,
}
*/

type LightClientUpdate struct {
}

type Slot uint64

type BeaconBlockHeader struct {
	Slot          Slot   `json:"slot"`
	ProposerIndex uint64 `json:"proposer_index"`
}
