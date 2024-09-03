package clientapi

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	local_spec "github.com/migalabs/goteth/pkg/spec"
)

func (s *APIClient) RequestBlobSidecars(slot phase0.Slot) ([]*local_spec.AgnosticBlobSidecar, error) {

	agnosticBlobs := make([]*local_spec.AgnosticBlobSidecar, 0)

	blobsResp, err := s.Api.BlobSidecars(s.ctx, &api.BlobSidecarsOpts{
		Block: fmt.Sprintf("%d", slot),
	})

	if err != nil {
		if response404(err.Error()) {
			return agnosticBlobs, nil
		}
		return nil, fmt.Errorf("could not retrieve blob sidecars for slot %d: %s", slot, err)
	}

	blobs := blobsResp.Data

	for _, item := range blobs {
		agnosticBlob := local_spec.NewAgnosticBlobFromAPI(slot, *item)
		agnosticBlobs = append(agnosticBlobs, agnosticBlob)
	}

	return agnosticBlobs, nil
}
