package metrics

import (
	"bytes"

	"math"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

type Phase0Metrics struct {
	baseMetrics StateMetricsBase
}

func NewPhase0Metrics(nextState *spec.AgnosticState, currentState *spec.AgnosticState, prevState *spec.AgnosticState) Phase0Metrics {

	phase0Obj := Phase0Metrics{}

	phase0Obj.InitBundle(nextState, currentState, prevState)
	phase0Obj.PreProcessBundle()

	return phase0Obj

}

func (p *Phase0Metrics) InitBundle(nextState *spec.AgnosticState,
	currentState *spec.AgnosticState,
	prevState *spec.AgnosticState) {
	p.baseMetrics.NextState = nextState
	p.baseMetrics.CurrentState = currentState
	p.baseMetrics.PrevState = prevState
	p.baseMetrics.MaxBlockRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	p.baseMetrics.MaxSlashingRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	p.baseMetrics.InclusionDelays = make([]int, len(p.baseMetrics.NextState.Validators))
	p.baseMetrics.MaxAttesterRewards = make(map[phase0.ValidatorIndex]phase0.Gwei)
	p.baseMetrics.CurrentNumAttestingVals = make([]bool, len(currentState.Validators))
}

func (p *Phase0Metrics) PreProcessBundle() {

	if !p.baseMetrics.PrevState.EmptyStateRoot() && !p.baseMetrics.CurrentState.EmptyStateRoot() {
		p.GetInclusionDelayDeltas()
		p.GetMaxAttComponentDeltas()
	}
}

func (p Phase0Metrics) GetMetricsBase() StateMetricsBase {
	return p.baseMetrics
}

// Processes attestations and fills several structs
func (p *Phase0Metrics) GetInclusionDelayDeltas() {

	prevAttestations := orderAttestationsBySlot(p.baseMetrics.CurrentState.PrevAttestations)

	for _, attestation := range prevAttestations {

		slot := attestation.Data.Slot            // Block that is being attested, not included
		committeeIndex := attestation.Data.Index // committee in the attested slot
		inclusionSlot := slot + attestation.InclusionDelay
		inclusionBlock, err := p.baseMetrics.GetBlockFromSlot(inclusionSlot)
		if err != nil {
			log.Fatal(err)
		}
		proposerIndex := inclusionBlock.ProposerIndex

		attValidatorIDs := p.baseMetrics.PrevState.EpochStructs.GetValList(slot, committeeIndex) // Beacon Committee
		attestingIndices := attestation.AggregationBits.BitIndices()                             // we only get the 1s, meaning the validator voted

		for _, index := range attestingIndices {
			attestingValIdx := attValidatorIDs[index]
			inclusionBlock.VotesIncluded += 1

			// if inclusion delay has not been set. Remember that attestations are order by slot asc
			if p.baseMetrics.InclusionDelays[attestingValIdx] == 0 {
				p.baseMetrics.InclusionDelays[attestingValIdx] = int(attestation.InclusionDelay)
				inclusionBlock.NewVotesIncluded += 1
				bestPossibleInclusionDelay := p.getMinInclusionDelayPossible(slot)

				// add correct flags and balances
				if p.IsCorrectSource() {
					p.baseMetrics.CurrentState.PrevEpochCorrectFlags[spec.AttSourceFlagIndex][attestingValIdx] = true
					p.baseMetrics.CurrentState.AttestingBalance[spec.AttSourceFlagIndex] += p.baseMetrics.CurrentState.Validators[attestingValIdx].EffectiveBalance

					// configure attester participation
					p.baseMetrics.CurrentNumAttestingVals[attestingValIdx] = true

					// add proposer reward
					proposerReward := p.GetProposerReward(attestingValIdx)
					p.baseMetrics.MaxBlockRewards[proposerIndex] += proposerReward
					inclusionBlock.ManualReward += proposerReward

					// add attester rewards
					maxAttesterReward := p.GetBaseReward(attestingValIdx) - proposerReward
					p.baseMetrics.MaxAttesterRewards[attestingValIdx] += maxAttesterReward / phase0.Gwei(bestPossibleInclusionDelay)

				}

				if p.IsCorrectTarget(*attestation) {
					p.baseMetrics.CurrentState.PrevEpochCorrectFlags[spec.AttTargetFlagIndex][attestingValIdx] = true
					p.baseMetrics.CurrentState.AttestingBalance[spec.AttTargetFlagIndex] += p.baseMetrics.CurrentState.Validators[attestingValIdx].EffectiveBalance
				}

				if p.IsCorrectHead(*attestation) {
					p.baseMetrics.CurrentState.PrevEpochCorrectFlags[spec.AttHeadFlagIndex][attestingValIdx] = true
					p.baseMetrics.CurrentState.AttestingBalance[spec.AttHeadFlagIndex] += p.baseMetrics.CurrentState.Validators[attestingValIdx].EffectiveBalance
				}
			}
		}
	}

	for valIdx, inclusionDelay := range p.baseMetrics.InclusionDelays {
		if inclusionDelay == 0 {
			p.baseMetrics.InclusionDelays[valIdx] = spec.SlotsPerEpoch + 1
		}
	}
}

func (p *Phase0Metrics) GetMaxAttComponentDeltas() {
	if p.baseMetrics.CurrentState.Epoch == 0 { // No rewards are applied at genesis
		return
	}
	for valIdx, validator := range p.baseMetrics.NextState.Validators {
		// if not in the list of validators or not active
		if !spec.IsActive(*validator, phase0.Epoch(p.baseMetrics.PrevState.Epoch)) {
			continue
		}

		baseReward := p.GetBaseReward(phase0.ValidatorIndex(valIdx))
		maxReward := phase0.Gwei(0)

		for i := range p.baseMetrics.CurrentState.PrevEpochCorrectFlags {

			previousAttestedBalance := p.baseMetrics.CurrentState.AttestingBalance[i]

			// participationRate per flag ==> previousAttestBalance / TotalActiveBalance
			singleReward := baseReward * (previousAttestedBalance / spec.EffectiveBalanceInc)

			// for each flag, we add baseReward * participationRate
			maxReward += singleReward / (p.baseMetrics.CurrentState.TotalActiveBalance / spec.EffectiveBalanceInc)
		}
		p.baseMetrics.MaxAttesterRewards[phase0.ValidatorIndex(valIdx)] += maxReward

	}
}

// TODO: review formulas
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#rewards-and-penalties-1
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#components-of-attestation-deltas
func (p Phase0Metrics) GetMaxReward(valIdx phase0.ValidatorIndex) (spec.ValidatorRewards, error) {

	maxReward := phase0.Gwei(0)

	proposerReward := p.baseMetrics.MaxBlockRewards[valIdx] // it is only the reward for the previous epoch participation

	maxReward += p.baseMetrics.MaxAttesterRewards[valIdx]
	maxReward += p.baseMetrics.MaxSlashingRewards[valIdx]
	maxReward += proposerReward

	result := spec.ValidatorRewards{
		ValidatorIndex:      valIdx,
		Epoch:               p.baseMetrics.NextState.Epoch,
		ValidatorBalance:    p.baseMetrics.CurrentState.Balances[valIdx],
		Reward:              p.baseMetrics.EpochReward(valIdx),
		MaxReward:           maxReward,
		AttestationReward:   p.baseMetrics.MaxAttesterRewards[valIdx],
		SyncCommitteeReward: 0,
		// AttSlot:              p.baseMetrics.PrevState.EpochStructs.ValidatorAttSlot[valIdx],
		MissingSource:        !p.baseMetrics.CurrentState.PrevEpochCorrectFlags[spec.AttSourceFlagIndex][valIdx],
		MissingTarget:        !p.baseMetrics.CurrentState.PrevEpochCorrectFlags[spec.AttTargetFlagIndex][valIdx],
		MissingHead:          !p.baseMetrics.CurrentState.PrevEpochCorrectFlags[spec.AttHeadFlagIndex][valIdx],
		Status:               p.baseMetrics.NextState.GetValStatus(valIdx),
		BaseReward:           p.GetBaseReward(valIdx),
		ProposerManualReward: proposerReward,
		ProposerApiReward:    0,
		InSyncCommittee:      false,
		InclusionDelay:       p.baseMetrics.InclusionDelays[valIdx],
	}
	return result, nil
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectSource() bool {
	epoch := phase0.Epoch(p.baseMetrics.CurrentState.Slot / spec.SlotsPerEpoch)
	if epoch == p.baseMetrics.CurrentState.Epoch || epoch == p.baseMetrics.PrevState.Epoch {
		return true
	}
	return false
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectTarget(attestation phase0.PendingAttestation) bool {
	target := attestation.Data.Target.Root

	slot := p.baseMetrics.PrevState.Slot / spec.SlotsPerEpoch
	slot = slot * spec.SlotsPerEpoch
	expected := p.baseMetrics.PrevState.BlockRoots[slot%spec.SlotsPerHistoricalRoot]

	res := bytes.Compare(target[:], expected[:])

	return res == 0 // if 0, then block roots are the same
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helper-functions-1
func (p Phase0Metrics) IsCorrectHead(attestation phase0.PendingAttestation) bool {
	head := attestation.Data.BeaconBlockRoot

	index := attestation.Data.Slot % spec.SlotsPerHistoricalRoot
	expected := p.baseMetrics.CurrentState.BlockRoots[index]

	res := bytes.Compare(head[:], expected[:])
	return res == 0 // if 0, then block roots are the same
}

// BaseReward = ( effectiveBalance * (BaseRewardFactor)/(BaseRewardsPerEpoch * sqrt(activeBalance)) )
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#helpers
func (p Phase0Metrics) GetBaseReward(valIdx phase0.ValidatorIndex) phase0.Gwei {

	var baseReward phase0.Gwei
	valEffectiveBalance := p.baseMetrics.CurrentState.Validators[valIdx].EffectiveBalance

	sqrt := math.Sqrt(float64(p.baseMetrics.CurrentState.TotalActiveBalance))
	denom := spec.BaseRewardPerEpoch * sqrt
	num := (valEffectiveBalance * spec.BaseRewardFactor)

	baseReward = phase0.Gwei(num) / phase0.Gwei(denom)

	return baseReward
}

func (p Phase0Metrics) getMinInclusionDelayPossible(slot phase0.Slot) int {

	result := 1
	for i := slot + 1; i <= (slot + phase0.Slot(spec.SlotsPerEpoch)); i++ {
		block, err := p.baseMetrics.GetBlockFromSlot(i)
		if err != nil {
			// was fatal
			log.Errorf("could not find best inclusion delay: %s", err)
		}

		if block.Proposed { // if there was a block proposed inside the inclusion window
			return result
		}
		result += 1
	}
	return result
}

func orderAttestationsBySlot(attestations []*phase0.PendingAttestation) []*phase0.PendingAttestation {
	orderedAttestations := make([]*phase0.PendingAttestation, 0)

	for i, attestation := range attestations {
		aInclusionSlot := attestation.Data.Slot + attestation.InclusionDelay
		var j int
		for j = 0; j < len(orderedAttestations); j++ {
			bInclusionSlot := attestations[j].Data.Slot + attestations[j].InclusionDelay
			if aInclusionSlot < bInclusionSlot {
				break
			}
		}

		reorg := append(orderedAttestations[:j], attestations[i])
		if j < len(orderedAttestations) {
			reorg = append(reorg, orderedAttestations[j:]...)
		}
		orderedAttestations = reorg
	}

	return orderedAttestations
}

func (p Phase0Metrics) GetProposerReward(attesterValIdx phase0.ValidatorIndex) phase0.Gwei {
	return phase0.Gwei(p.GetBaseReward(attesterValIdx) / spec.ProposerRewardQuotient)
}
