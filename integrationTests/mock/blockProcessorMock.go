package mock

import (
	"time"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/marshal"
)

// BlockProcessorMock mocks the implementation for a blockProcessor
type BlockProcessorMock struct {
	NrCommitBlockCalled                     uint32
	Marshalizer                             marshal.Marshalizer
	ProcessBlockCalled                      func(blockChain data.ChainHandler, header data.HeaderHandler, body data.BodyHandler, haveTime func() time.Duration) error
	CommitBlockCalled                       func(blockChain data.ChainHandler, header data.HeaderHandler, body data.BodyHandler) error
	RevertAccountStateCalled                func()
	CreateBlockCalled                       func(initialHdrData data.HeaderHandler, haveTime func() bool) (data.BodyHandler, error)
	RestoreBlockIntoPoolsCalled             func(header data.HeaderHandler, body data.BodyHandler) error
	ApplyBodyToHeaderCalled                 func(header data.HeaderHandler, body data.BodyHandler) error
	MarshalizedDataToBroadcastCalled        func(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error)
	DecodeBlockBodyCalled                   func(dta []byte) data.BodyHandler
	DecodeBlockHeaderCalled                 func(dta []byte) data.HeaderHandler
	AddLastNotarizedHdrCalled               func(shardId uint32, processedHdr data.HeaderHandler)
	SetConsensusDataCalled                  func(randomness []byte, round uint64, epoch uint32, shardId uint32)
	CreateNewHeaderCalled                   func() data.HeaderHandler
	RevertStateToBlockCalled                func(header data.HeaderHandler) error
	RestoreLastNotarizedHrdsToGenesisCalled func()
}

func (blProcMock *BlockProcessorMock) RestoreLastNotarizedHrdsToGenesis() {
	if blProcMock.RestoreLastNotarizedHrdsToGenesisCalled != nil {
		blProcMock.RestoreLastNotarizedHrdsToGenesisCalled()
	}
}

// ProcessBlock mocks pocessing a block
func (blProcMock *BlockProcessorMock) ProcessBlock(blockChain data.ChainHandler, header data.HeaderHandler, body data.BodyHandler, haveTime func() time.Duration) error {
	return blProcMock.ProcessBlockCalled(blockChain, header, body, haveTime)
}

func (blProcMock *BlockProcessorMock) ApplyProcessedMiniBlocks(miniBlocks map[string]map[string]struct{}) {

}

// CommitBlock mocks the commit of a block
func (blProcMock *BlockProcessorMock) CommitBlock(blockChain data.ChainHandler, header data.HeaderHandler, body data.BodyHandler) error {
	return blProcMock.CommitBlockCalled(blockChain, header, body)
}

// RevertAccountState mocks revert of the accounts state
func (blProcMock *BlockProcessorMock) RevertAccountState() {
	blProcMock.RevertAccountStateCalled()
}

func (blProcMock *BlockProcessorMock) CreateNewHeader() data.HeaderHandler {
	return blProcMock.CreateNewHeaderCalled()
}

// CreateTxBlockBody mocks the creation of a transaction block body
func (blProcMock *BlockProcessorMock) CreateBlockBody(initialHdrData data.HeaderHandler, haveTime func() bool) (data.BodyHandler, error) {
	return blProcMock.CreateBlockCalled(initialHdrData, haveTime)
}

func (blProcMock *BlockProcessorMock) RestoreBlockIntoPools(header data.HeaderHandler, body data.BodyHandler) error {
	return blProcMock.RestoreBlockIntoPoolsCalled(header, body)
}

func (blProcMock BlockProcessorMock) ApplyBodyToHeader(header data.HeaderHandler, body data.BodyHandler) error {
	return blProcMock.ApplyBodyToHeaderCalled(header, body)
}

// RevertStateToBlock recreates the state tries to the root hashes indicated by the provided header
func (blProcMock *BlockProcessorMock) RevertStateToBlock(header data.HeaderHandler) error {
	if blProcMock.RevertStateToBlockCalled != nil {
		return blProcMock.RevertStateToBlock(header)
	}
	return nil
}

func (blProcMock *BlockProcessorMock) SetNumProcessedObj(numObj uint64) {

}

func (blProcMock BlockProcessorMock) MarshalizedDataToBroadcast(header data.HeaderHandler, body data.BodyHandler) (map[uint32][]byte, map[string][][]byte, error) {
	return blProcMock.MarshalizedDataToBroadcastCalled(header, body)
}

// DecodeBlockBody method decodes block body from a given byte array
func (blProcMock BlockProcessorMock) DecodeBlockBody(dta []byte) data.BodyHandler {
	if dta == nil {
		return nil
	}

	var body block.Body

	err := blProcMock.Marshalizer.Unmarshal(&body, dta)
	if err != nil {
		return nil
	}

	return body
}

// DecodeBlockHeader method decodes block header from a given byte array
func (blProcMock BlockProcessorMock) DecodeBlockHeader(dta []byte) data.HeaderHandler {
	if dta == nil {
		return nil
	}

	var header block.Header

	err := blProcMock.Marshalizer.Unmarshal(&header, dta)
	if err != nil {
		return nil
	}

	return &header
}

func (blProcMock BlockProcessorMock) AddLastNotarizedHdr(shardId uint32, processedHdr data.HeaderHandler) {
	blProcMock.AddLastNotarizedHdrCalled(shardId, processedHdr)
}

func (blProcMock BlockProcessorMock) SetConsensusData(randomness []byte, round uint64, epoch uint32, shardId uint32) {
	if blProcMock.SetConsensusDataCalled != nil {
		blProcMock.SetConsensusDataCalled(randomness, round, epoch, shardId)
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (blProcMock *BlockProcessorMock) IsInterfaceNil() bool {
	if blProcMock == nil {
		return true
	}
	return false
}
