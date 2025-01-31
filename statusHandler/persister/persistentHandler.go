package persister

import (
	"sync"
	"time"

	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/storage"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
)

var log = logger.GetOrCreate("statusHandler/persister")

// PersistentStatusHandler is a status handler that will save metrics in storage
type PersistentStatusHandler struct {
	store                    storage.Storer
	persistentMetrics        *sync.Map
	marshalizer              marshal.Marshalizer
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter
	startSaveInStorage       bool
}

// NewPersistentStatusHandler will return an instance of the persistent status handler
func NewPersistentStatusHandler(
	marshalizer marshal.Marshalizer,
	uint64ByteSliceConverter typeConverters.Uint64ByteSliceConverter,
) (*PersistentStatusHandler, error) {
	if check.IfNil(marshalizer) {
		return nil, process.ErrNilMarshalizer
	}
	if check.IfNil(uint64ByteSliceConverter) {
		return nil, process.ErrNilUint64Converter
	}

	psh := new(PersistentStatusHandler)
	psh.store = storageUnit.NewNilStorer()
	psh.uint64ByteSliceConverter = uint64ByteSliceConverter
	psh.marshalizer = marshalizer
	psh.persistentMetrics = &sync.Map{}
	psh.initMap()

	go func() {
		time.Sleep(time.Second)

		psh.startSaveInStorage = true
	}()

	return psh, nil
}

func (psh *PersistentStatusHandler) initMap() {
	initUint := uint64(0)

	psh.persistentMetrics.Store(core.MetricCountConsensus, initUint)
	psh.persistentMetrics.Store(core.MetricCountConsensusAcceptedBlocks, initUint)
	psh.persistentMetrics.Store(core.MetricCountAcceptedBlocks, initUint)
	psh.persistentMetrics.Store(core.MetricCountLeader, initUint)
	psh.persistentMetrics.Store(core.MetricNumProcessedTxs, initUint)
	psh.persistentMetrics.Store(core.MetricNumShardHeadersProcessed, initUint)
	psh.persistentMetrics.Store(core.MetricNonce, initUint)
}

// SetStorage will set storage for persistent status handler
func (psh *PersistentStatusHandler) SetStorage(store storage.Storer) error {
	if check.IfNil(store) {
		return process.ErrNilStorage
	}

	psh.store = store

	return nil
}

func (psh *PersistentStatusHandler) saveMetricsInDb(nonce uint64) {
	metricsMap := make(map[string]interface{})
	psh.persistentMetrics.Range(func(key, value interface{}) bool {
		keyString, ok := key.(string)
		if !ok {
			return false
		}
		metricsMap[keyString] = value
		return true
	})

	statusMetricsBytes, err := psh.marshalizer.Marshal(metricsMap)
	if err != nil {
		log.Debug("cannot marshal metrics map",
			"error", err)
		return
	}

	nonceBytes := psh.uint64ByteSliceConverter.ToByteSlice(nonce)
	err = psh.store.Put(nonceBytes, statusMetricsBytes)
	if err != nil {
		log.Debug("cannot save metrics map in storage",
			"error", err)
		return
	}
}

// SetInt64Value method - will update the value for a key
func (psh *PersistentStatusHandler) SetInt64Value(key string, value int64) {
	if _, ok := psh.persistentMetrics.Load(key); !ok {
		return
	}

	psh.persistentMetrics.Store(key, value)
}

// SetUInt64Value method - will update the value for a key
func (psh *PersistentStatusHandler) SetUInt64Value(key string, value uint64) {
	valueFromMapI, ok := psh.persistentMetrics.Load(key)
	if !ok {
		return
	}

	psh.persistentMetrics.Store(key, value)

	//metrics wil be saved in storage every time when a block is committed successfully
	if key != core.MetricNonce {
		return
	}

	valueFromMap := GetUint64(valueFromMapI)
	if value < valueFromMap {
		return
	}

	if !psh.startSaveInStorage {
		return
	}

	psh.saveMetricsInDb(value)
}

// SetStringValue method - will update the value of a key
func (psh *PersistentStatusHandler) SetStringValue(key string, value string) {
	if _, ok := psh.persistentMetrics.Load(key); !ok {
		return
	}

	psh.persistentMetrics.Store(key, value)
}

// Increment - will increment the value of a key
func (psh *PersistentStatusHandler) Increment(key string) {
	psh.AddUint64(key, 1)
}

// AddUint64 - will increase the value of a key with a value
func (psh *PersistentStatusHandler) AddUint64(key string, value uint64) {
	keyValueI, ok := psh.persistentMetrics.Load(key)
	if !ok {
		return
	}

	keyValue, ok := keyValueI.(uint64)
	if !ok {
		return
	}

	keyValue += value
	psh.persistentMetrics.Store(key, keyValue)
}

// Decrement - will decrement the value of a key
func (psh *PersistentStatusHandler) Decrement(key string) {
	keyValueI, ok := psh.persistentMetrics.Load(key)
	if !ok {
		return
	}

	keyValue, ok := keyValueI.(uint64)
	if !ok {
		return
	}
	if keyValue == 0 {
		return
	}

	keyValue--
	psh.persistentMetrics.Store(key, keyValue)
}

// Close method - won't do anything
func (psh *PersistentStatusHandler) Close() {
}

// IsInterfaceNil returns true if there is no value under the interface
func (psh *PersistentStatusHandler) IsInterfaceNil() bool {
	return psh == nil
}
