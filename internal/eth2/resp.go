package eth2

type CommonData struct {
	StatusCode          int         `json:"statusCode"`
	Error               string      `json:"error"`
	Message             string      `json:"message"`
	Data                interface{} `json:"data"`
	ExecutionOptimistic bool        `json:"execution_optimistic"`
}

type BeaconHeadersResp struct {
	Data                BeaconHeadersData `json:"data"`
	ExecutionOptimistic bool              `json:"execution_optimistic"`
}

type FinalityUpdateResp struct {
	Data FinalityUpdateData `json:"data"`
}

type BlocksResp struct {
	Data                BlockData `json:"data"`
	ExecutionOptimistic bool      `json:"execution_optimistic"`
}

type LightClientUpdatesResp struct {
	Data []LightClientUpdatesData `json:"data"`
}

type Message struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type Header struct {
	Message   Message `json:"message"`
	Signature string  `json:"signature"`
}

type BeaconHeadersData struct {
	Root      string `json:"root"`
	Canonical bool   `json:"canonical"`
	Header    Header `json:"header"`
}

type AttestedHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type FinalizedHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type SyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`
	SyncCommitteeSignature string `json:"sync_committee_signature"`
}

type FinalityUpdateData struct {
	AttestedHeader  AttestedHeader  `json:"attested_header"`
	FinalizedHeader FinalizedHeader `json:"finalized_header"`
	FinalityBranch  []string        `json:"finality_branch"`
	SyncAggregate   SyncAggregate   `json:"sync_aggregate"`
	SignatureSlot   string          `json:"signature_slot"`
}

type Eth1Data struct {
	DepositRoot  string `json:"deposit_root"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash"`
}

type Source struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

type Target struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

type Attestations struct {
	AggregationBits string          `json:"aggregation_bits"`
	Data            AttestationData `json:"data"`
	Signature       string          `json:"signature"`
}

type Body struct {
	RandaoReveal      string         `json:"randao_reveal"`
	Eth1Data          Eth1Data       `json:"eth1_data"`
	Graffiti          string         `json:"graffiti"`
	ProposerSlashings []interface{}  `json:"proposer_slashings"`
	AttesterSlashings []interface{}  `json:"attester_slashings"`
	Attestations      []Attestations `json:"attestations"`
	Deposits          []interface{}  `json:"deposits"`
	VoluntaryExits    []interface{}  `json:"voluntary_exits"`
}

type BlocksMessage struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	Body          Body   `json:"body"`
}

type BlockData struct {
	Message   BlocksMessage `json:"message"`
	Signature string        `json:"signature"`
}

type NextSyncCommittee struct {
	Pubkeys         []string `json:"pubkeys"`
	AggregatePubkey string   `json:"aggregate_pubkey"`
}

type LightClientUpdatesData struct {
	AttestedHeader          AttestedHeader    `json:"attested_header"`
	NextSyncCommittee       NextSyncCommittee `json:"next_sync_committee"`
	NextSyncCommitteeBranch []string          `json:"next_sync_committee_branch"`
	FinalizedHeader         FinalizedHeader   `json:"finalized_header"`
	FinalityBranch          []string          `json:"finality_branch"`
	SyncAggregate           SyncAggregate     `json:"sync_aggregate"`
	SignatureSlot           string            `json:"signature_slot"`
}

type AttestationData struct {
	BeaconBlockRoot string `json:"beacon_block_root"`
	Index           string `json:"index"`
	Slot            string `json:"slot"`
	Source          Source `json:"source"`
	Target          Target `json:"target"`
}
