package db

import (
	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	attestationsTable      = "t_attestations"
	insertAttestationQuery = `
	INSERT INTO %s (
		f_timestamp,
		f_epoch, 
		f_slot,
		f_attestation_index,
		f_attestation_slot,
		f_attestation_beacon_block_root,
		f_attestation_source_epoch,
		f_attestation_source_root,
		f_attestation_target_epoch,
		f_attestation_target_root)
		VALUES`

	deleteAttestationQuery = `
		DELETE FROM %s
		WHERE f_slot = $1;
`
)

func attestationInput(attestations []spec.Attestation) proto.Input {
	// one object per column
	var (
		f_timestamp                     proto.ColUInt64
		f_epoch                         proto.ColUInt64
		f_slot                          proto.ColUInt64
		f_attestation_index             proto.ColUInt64
		f_attestation_slot              proto.ColUInt64
		f_attestation_beacon_block_root proto.ColStr
		f_attestation_source_epoch      proto.ColUInt64
		f_attestation_source_root       proto.ColStr
		f_attestation_target_epoch      proto.ColUInt64
		f_attestation_target_root       proto.ColStr
	)

	for _, attestation := range attestations {

		if attestation.Attestation != nil && attestation.Attestation.Data != nil {

			f_timestamp.Append(uint64(attestation.Timestamp))
			f_epoch.Append(uint64(attestation.Slot / spec.SlotsPerEpoch))
			f_slot.Append(uint64(attestation.Slot))

			f_attestation_index.Append(uint64(attestation.Attestation.Data.Index))
			f_attestation_slot.Append(uint64(attestation.Attestation.Data.Slot))
			f_attestation_beacon_block_root.Append(attestation.Attestation.Data.BeaconBlockRoot.String())

			if attestation.Attestation.Data.Source != nil {
				f_attestation_source_epoch.Append(uint64(attestation.Attestation.Data.Source.Epoch))
				f_attestation_source_root.Append(attestation.Attestation.Data.Source.Root.String())
			}

			if attestation.Attestation.Data.Target != nil {
				f_attestation_target_epoch.Append(uint64(attestation.Attestation.Data.Target.Epoch))
				f_attestation_target_root.Append(attestation.Attestation.Data.Target.Root.String())
			}
		}
	}

	return proto.Input{
		{Name: "f_attestation_slot", Data: f_attestation_slot},
		{Name: "f_timestamp", Data: f_timestamp},
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_slot", Data: f_slot},
		{Name: "f_attestation_index", Data: f_attestation_index},
		{Name: "f_attestation_beacon_block_root", Data: f_attestation_beacon_block_root},
		{Name: "f_attestation_source_epoch", Data: f_attestation_source_epoch},
		{Name: "f_attestation_source_root", Data: f_attestation_source_root},
		{Name: "f_attestation_target_epoch", Data: f_attestation_target_epoch},
		{Name: "f_attestation_target_root", Data: f_attestation_target_root},
	}
}

func (s *DBService) DeleteAttestationsMetrics(slot phase0.Slot) error {

	err := s.Delete(DeletableObject{
		query: deleteAttestationQuery,
		table: attestationsTable,
		args:  []any{slot},
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *DBService) PersistAttestations(data []spec.AgnosticBlock) error {

	attestations := make([]spec.Attestation, 0)
	for _, block := range data {
		for _, attestation := range block.Attestations {
			att := spec.Attestation{}
			att.Slot = phase0.Slot(block.Slot)
			att.Timestamp = block.ExecutionPayload.Timestamp
			att.Attestation = attestation
			attestations = append(attestations, att)
		}
	}

	persistObj := PersistableObject[spec.Attestation]{
		input: attestationInput,
		table: attestationsTable,
		query: insertAttestationQuery,
	}

	for _, att := range attestations {
		persistObj.Append(att)
	}

	err := p.Persist(persistObj.ExportPersist())
	if err != nil {
		log.Errorf("error persisting attestations: %s", err.Error())
	}
	return err
}
