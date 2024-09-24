package analyzer

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

var (
	slotProcesserTag = "slot="
)

func (s *ChainAnalyzer) ProcessBlock(slot phase0.Slot) {
	if !s.metrics.Block {
		return
	}
	routineKey := fmt.Sprintf("%s%d", slotProcesserTag, slot)
	s.processerBook.Acquire(routineKey) // register a new slot to process, good for monitoring

	block := s.downloadCache.BlockHistory.Wait(SlotTo[uint64](slot))

	agnosticBlock := []spec.AgnosticBlock{*block}

	err := s.dbClient.PersistBlocks(agnosticBlock)
	if err != nil {
		log.Errorf("error persisting blocks: %s", err.Error())
	}

	errAtt := s.dbClient.PersistAttestations(agnosticBlock)
	if errAtt != nil {
		log.Errorf("error persisting attestations: %s", errAtt.Error())
	}

	var withdrawals []spec.Withdrawal
	for _, item := range block.ExecutionPayload.Withdrawals {
		withdrawals = append(withdrawals, spec.Withdrawal{
			Slot:           block.Slot,
			Index:          item.Index,
			ValidatorIndex: item.ValidatorIndex,
			Address:        item.Address,
			Amount:         item.Amount,
		})
	}

	err = s.dbClient.PersistWithdrawals(withdrawals)
	if err != nil {
		log.Errorf("error persisting withdrawals: %s", err.Error())
	}

	if s.metrics.Transactions {
		s.processTransactions(block)
		s.processBlobSidecars(block, block.ExecutionPayload.AgnosticTransactions)
	}
	s.processerBook.FreePage(routineKey)
}

func (s *ChainAnalyzer) processTransactions(block *spec.AgnosticBlock) {

	txs, err := s.cli.GetBlockTransactions(*block)
	if err != nil {
		log.Errorf("error getting slot %d transactions: %s", block.Slot, err.Error())
	}
	block.ExecutionPayload.AgnosticTransactions = txs

	err = s.dbClient.PersistTransactions(txs)
	if err != nil {
		log.Errorf("error persisting transactions: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processBlobSidecars(block *spec.AgnosticBlock, txs []spec.AgnosticTransaction) {

	persistable := make([]*spec.AgnosticBlobSidecar, 0)

	blobs, err := s.cli.RequestBlobSidecars(block.Slot)

	if err != nil {
		log.Warningf("blob sidecards for slot %d: %s", block.Slot, err)
	} else {
		log.Infof("fetched blob sidecards for slot %d", block.Slot)
		if len(blobs) > 0 {
			for _, blob := range blobs {
				blob.GetTxHash(txs)
				persistable = append(persistable, blob)
			}
			s.dbClient.PersistBlobSidecars(persistable)
		}
	}
}
