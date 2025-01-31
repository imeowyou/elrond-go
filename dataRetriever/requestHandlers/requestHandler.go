package requestHandlers

import (
	"fmt"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/core/partitioning"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/logger"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type resolverRequestHandler struct {
	resolversFinder       dataRetriever.ResolversFinder
	requestedItemsHandler dataRetriever.RequestedItemsHandler
	txRequestTopic        string
	scrRequestTopic       string
	rewardTxRequestTopic  string
	mbRequestTopic        string
	shardHdrRequestTopic  string
	metaHdrRequestTopic   string
	isMetaChain           bool
	maxTxsToRequest       int
	sweepTime             time.Time
}

var log = logger.GetOrCreate("dataretriever/requesthandlers")

// NewShardResolverRequestHandler creates a requestHandler interface implementation with request functions
func NewShardResolverRequestHandler(
	finder dataRetriever.ResolversFinder,
	requestedItemsHandler dataRetriever.RequestedItemsHandler,
	txRequestTopic string,
	scrRequestTopic string,
	rewardTxRequestTopic string,
	mbRequestTopic string,
	shardHdrRequestTopic string,
	metaHdrRequestTopic string,
	maxTxsToRequest int,
) (*resolverRequestHandler, error) {

	if check.IfNil(finder) {
		return nil, dataRetriever.ErrNilResolverFinder
	}
	if check.IfNil(requestedItemsHandler) {
		return nil, dataRetriever.ErrNilRequestedItemsHandler
	}
	if len(txRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyTxRequestTopic
	}
	if len(scrRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyScrRequestTopic
	}
	if len(rewardTxRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyRewardTxRequestTopic
	}
	if len(mbRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyMiniBlockRequestTopic
	}
	if len(shardHdrRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyShardHeaderRequestTopic
	}
	if len(metaHdrRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyMetaHeaderRequestTopic
	}
	if maxTxsToRequest < 1 {
		return nil, dataRetriever.ErrInvalidMaxTxRequest
	}

	rrh := &resolverRequestHandler{
		resolversFinder:       finder,
		requestedItemsHandler: requestedItemsHandler,
		txRequestTopic:        txRequestTopic,
		mbRequestTopic:        mbRequestTopic,
		shardHdrRequestTopic:  shardHdrRequestTopic,
		metaHdrRequestTopic:   metaHdrRequestTopic,
		scrRequestTopic:       scrRequestTopic,
		rewardTxRequestTopic:  rewardTxRequestTopic,
		isMetaChain:           false,
		maxTxsToRequest:       maxTxsToRequest,
	}

	rrh.sweepTime = time.Now()

	return rrh, nil
}

// NewMetaResolverRequestHandler creates a requestHandler interface implementation with request functions
func NewMetaResolverRequestHandler(
	finder dataRetriever.ResolversFinder,
	requestedItemsHandler dataRetriever.RequestedItemsHandler,
	shardHdrRequestTopic string,
	metaHdrRequestTopic string,
	txRequestTopic string,
	scrRequestTopic string,
	mbRequestTopic string,
	maxTxsToRequest int,
) (*resolverRequestHandler, error) {

	if check.IfNil(finder) {
		return nil, dataRetriever.ErrNilResolverFinder
	}
	if check.IfNil(requestedItemsHandler) {
		return nil, dataRetriever.ErrNilRequestedItemsHandler
	}
	if len(shardHdrRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyShardHeaderRequestTopic
	}
	if len(metaHdrRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyMetaHeaderRequestTopic
	}
	if len(txRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyTxRequestTopic
	}
	if len(scrRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyScrRequestTopic
	}
	if len(mbRequestTopic) == 0 {
		return nil, dataRetriever.ErrEmptyMiniBlockRequestTopic
	}
	if maxTxsToRequest < 1 {
		return nil, dataRetriever.ErrInvalidMaxTxRequest
	}

	rrh := &resolverRequestHandler{
		resolversFinder:       finder,
		requestedItemsHandler: requestedItemsHandler,
		shardHdrRequestTopic:  shardHdrRequestTopic,
		metaHdrRequestTopic:   metaHdrRequestTopic,
		txRequestTopic:        txRequestTopic,
		mbRequestTopic:        mbRequestTopic,
		scrRequestTopic:       scrRequestTopic,
		isMetaChain:           true,
		maxTxsToRequest:       maxTxsToRequest,
	}

	return rrh, nil
}

// RequestTransaction method asks for transactions from the connected peers
func (rrh *resolverRequestHandler) RequestTransaction(destShardID uint32, txHashes [][]byte) {
	rrh.requestByHashes(destShardID, txHashes, rrh.txRequestTopic)
}

func (rrh *resolverRequestHandler) requestByHashes(destShardID uint32, hashes [][]byte, topic string) {
	unrequestedHashes := rrh.getUnrequestedHashes(hashes)
	log.Trace("requesting transactions from network",
		"num txs", len(unrequestedHashes),
		"topic", topic,
		"shard", destShardID,
	)
	resolver, err := rrh.resolversFinder.CrossShardResolver(topic, destShardID)
	if err != nil {
		log.Error("missing resolver",
			"topic", topic,
			"shard", destShardID,
		)
		return
	}

	txResolver, ok := resolver.(HashSliceResolver)
	if !ok {
		log.Debug("wrong assertion type when creating transaction resolver")
		return
	}

	go func() {
		dataSplit := &partitioning.DataSplit{}
		sliceBatches, err := dataSplit.SplitDataInChunks(unrequestedHashes, rrh.maxTxsToRequest)
		if err != nil {
			log.Debug("requesting transactions", "error", err.Error())
			return
		}

		for _, batch := range sliceBatches {
			err = txResolver.RequestDataFromHashArray(batch)
			if err != nil {
				log.Debug("requesting tx batch", "error", err.Error())
			}
		}
	}()
}

// RequestUnsignedTransactions method asks for unsigned transactions from the connected peers
func (rrh *resolverRequestHandler) RequestUnsignedTransactions(destShardID uint32, scrHashes [][]byte) {
	rrh.requestByHashes(destShardID, scrHashes, rrh.scrRequestTopic)
}

// RequestRewardTransactions requests for reward transactions from the connected peers
func (rrh *resolverRequestHandler) RequestRewardTransactions(destShardId uint32, rewardTxHashes [][]byte) {
	rrh.requestByHashes(destShardId, rewardTxHashes, rrh.rewardTxRequestTopic)
}

// RequestMiniBlock method asks for miniblocks from the connected peers
func (rrh *resolverRequestHandler) RequestMiniBlock(destShardID uint32, miniblockHash []byte) {
	rrh.sweepIfNeeded()

	if rrh.requestedItemsHandler.Has(string(miniblockHash)) {
		log.Trace("item already requested",
			"key", miniblockHash)
		return
	}

	log.Trace("requesting miniblock from network",
		"hash", miniblockHash,
		"shard", destShardID,
		"topic", rrh.mbRequestTopic,
	)

	resolver, err := rrh.resolversFinder.CrossShardResolver(rrh.mbRequestTopic, destShardID)
	if err != nil {
		log.Error("missing resolver",
			"topic", rrh.mbRequestTopic,
			"shard", destShardID,
		)
		return
	}

	err = resolver.RequestDataFromHash(miniblockHash)
	if err != nil {
		log.Debug(err.Error())
		return
	}

	err = rrh.requestedItemsHandler.Add(string(miniblockHash))
	if err != nil {
		log.Trace("add requested item with error",
			"error", err.Error(),
			"key", miniblockHash)
	}
}

// RequestHeader method asks for header from the connected peers
func (rrh *resolverRequestHandler) RequestHeader(destShardID uint32, hash []byte) {
	rrh.sweepIfNeeded()

	if rrh.requestedItemsHandler.Has(string(hash)) {
		log.Trace("item already requested",
			"key", hash)
		return
	}

	//TODO: Refactor this class and create specific methods for requesting shard or meta data
	var baseTopic string
	if destShardID == sharding.MetachainShardId {
		baseTopic = rrh.metaHdrRequestTopic
	} else {
		baseTopic = rrh.shardHdrRequestTopic
	}

	log.Trace("requesting by hash",
		"topic", baseTopic,
		"shard", destShardID,
		"hash", hash,
	)

	var resolver dataRetriever.Resolver
	var err error

	if destShardID == sharding.MetachainShardId {
		resolver, err = rrh.resolversFinder.MetaChainResolver(baseTopic)
	} else {
		resolver, err = rrh.resolversFinder.CrossShardResolver(baseTopic, destShardID)
	}

	if err != nil {
		log.Error("missing resolver",
			"topic", baseTopic,
			"shard", destShardID,
		)
		return
	}

	err = resolver.RequestDataFromHash(hash)
	if err != nil {
		log.Debug("RequestDataFromHash", "error", err.Error())
		return
	}

	err = rrh.requestedItemsHandler.Add(string(hash))
	if err != nil {
		log.Trace("add requested item with error",
			"error", err.Error(),
			"key", hash)
	}
}

// RequestHeaderByNonce method asks for transactions from the connected peers
func (rrh *resolverRequestHandler) RequestHeaderByNonce(destShardID uint32, nonce uint64) {
	rrh.sweepIfNeeded()

	key := fmt.Sprintf("%d-%d", destShardID, nonce)
	if rrh.requestedItemsHandler.Has(key) {
		log.Trace("item already requested",
			"key", key)
		return
	}

	var err error
	var resolver dataRetriever.Resolver
	var topic string
	if rrh.isMetaChain {
		topic = rrh.shardHdrRequestTopic
		resolver, err = rrh.resolversFinder.CrossShardResolver(topic, destShardID)
	} else {
		topic = rrh.metaHdrRequestTopic
		resolver, err = rrh.resolversFinder.MetaChainResolver(topic)
	}

	if err != nil {
		log.Debug("missing resolver",
			"topic", topic,
			"shard", destShardID,
		)
		return
	}

	headerResolver, ok := resolver.(dataRetriever.HeaderResolver)
	if !ok {
		log.Debug("resolver is not a header resolver",
			"topic", topic,
			"shard", destShardID,
		)
		return
	}

	err = headerResolver.RequestDataFromNonce(nonce)
	if err != nil {
		log.Debug("RequestDataFromNonce", "error", err.Error())
		return
	}

	err = rrh.requestedItemsHandler.Add(key)
	if err != nil {
		log.Trace("add requested item with error",
			"error", err.Error(),
			"key", key)
	}
}

// IsInterfaceNil returns true if there is no value under the interface
func (rrh *resolverRequestHandler) IsInterfaceNil() bool {
	if rrh == nil {
		return true
	}
	return false
}

func (rrh *resolverRequestHandler) getUnrequestedHashes(hashes [][]byte) [][]byte {
	unrequestedHashes := make([][]byte, 0)

	rrh.sweepIfNeeded()

	for _, hash := range hashes {
		if !rrh.requestedItemsHandler.Has(string(hash)) {
			unrequestedHashes = append(unrequestedHashes, hash)
			err := rrh.requestedItemsHandler.Add(string(hash))
			if err != nil {
				log.Trace("add requested item with error",
					"error", err.Error(),
					"key", hash)
			}
		}
	}

	return unrequestedHashes
}

func (rrh *resolverRequestHandler) sweepIfNeeded() {
	if time.Since(rrh.sweepTime) <= time.Second {
		return
	}

	rrh.sweepTime = time.Now()
	rrh.requestedItemsHandler.Sweep()
}
