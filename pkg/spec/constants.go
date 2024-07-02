package spec

const (
	MainnetGenesis = 1606824023
	SepoliaGenesis = 1655733600
	HoleskyGenesis = 1695902400
	GnosisGenesis = 1638968400
)

// Specific Constants for Mainnet configuration
const (
    MainnetBaseRewardFactor            = 64
    MainnetSlotsPerEpoch               = 32
    MainnetSlotSeconds                 = 12
    MainnetEpochSlots                  = 32
)

// Specific Constants for Gnosis configuration
const (
    GnosisBaseRewardFactor            = 25
    GnosisSlotsPerEpoch               = 16
    GnosisSlotSeconds                 = 5
    GnosisEpochSlots                  = 16
)


/*
Phase0
*/

const (
	MaxEffectiveInc             = 32
	BaseRewardPerEpoch          = 4
	EffectiveBalanceInc         = 1000000000
	ProposerRewardQuotient      = 8
	SlotsPerHistoricalRoot      = 8192
	WhistleBlowerRewardQuotient = 512
	MinInclusionDelay           = 1

	AttSourceFlagIndex = 0
	AttTargetFlagIndex = 1
	AttHeadFlagIndex   = 2
)
// make this dynamic... fixed to Gnosis for now
const (
    BaseRewardFactor            = 25
    SlotsPerEpoch               = 16
    SlotSeconds                 = 5
    EpochSlots                  = 16
)

/*
Altair
*/
const (
	// spec weight constants
	TimelySourceWeight = 14
	TimelyTargetWeight = 26
	TimelyHeadWeight   = 14

	SyncRewardWeight  = 2
	ProposerWeight    = 8
	WeightDenominator = 64
	SyncCommitteeSize = 512
)

var (
	ParticipatingFlagsWeight = [3]int{TimelySourceWeight, TimelyTargetWeight, TimelyHeadWeight}
)

type ModelType int8

const (
	BlockModel ModelType = iota
	BlockDropModel
	OrphanModel
	EpochModel
	EpochDropModel
	PoolSummaryModel
	ProposerDutyModel
	ProposerDutyDropModel
	ValidatorLastStatusModel
	ValidatorRewardsModel
	ValidatorRewardDropModel
	WithdrawalModel
	WithdrawalDropModel
	TransactionsModel
	TransactionDropModel
	ReorgModel
	FinalizedCheckpointModel
	HeadEventModel
)

type ValidatorStatus int8

const (
	QUEUE_STATUS ValidatorStatus = iota
	ACTIVE_STATUS
	EXIT_STATUS
	SLASHED_STATUS
	NUMBER_OF_STATUS // Add new status before this
)
