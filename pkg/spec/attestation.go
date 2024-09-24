package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// TODO: review
type Attestation struct {
	Slot        phase0.Slot
	Timestamp   uint64
	Attestation *phase0.Attestation
}

func (f Attestation) Type() ModelType {
	return AttestationModel
}
