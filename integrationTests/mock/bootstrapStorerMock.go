package mock

import "github.com/ElrondNetwork/elrond-go/process/block/bootstrapStorage"

type BoostrapStorerMock struct {
	PutCalled             func(round int64, bootData bootstrapStorage.BootstrapData) error
	GetCalled             func(round int64) (bootstrapStorage.BootstrapData, error)
	GetHighestRoundCalled func() int64
}

func (bsm *BoostrapStorerMock) Put(round int64, bootData bootstrapStorage.BootstrapData) error {
	return bsm.PutCalled(round, bootData)
}

func (bsm *BoostrapStorerMock) Get(round int64) (bootstrapStorage.BootstrapData, error) {
	if bsm.GetCalled == nil {
		return bootstrapStorage.BootstrapData{}, bootstrapStorage.ErrNilMarshalizer
	}
	return bsm.GetCalled(round)
}

func (bsm *BoostrapStorerMock) GetHighestRound() int64 {
	if bsm.GetHighestRoundCalled == nil {
		return 0
	}
	return bsm.GetHighestRoundCalled()
}

func (bsm *BoostrapStorerMock) IsInterfaceNil() bool {
	return bsm == nil
}

func (bsm *BoostrapStorerMock) SaveLastRound(round int64) error {
	return nil
}
