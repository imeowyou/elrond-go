package sync

import (
	"fmt"
	"math"
	"time"

	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/typeConverters/uint64ByteSlice"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/factory"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-go/statusHandler"
	"github.com/ElrondNetwork/elrond-go/storage"
)

// MetaBootstrap implements the bootstrap mechanism
type MetaBootstrap struct {
	*baseBootstrap
}

// NewMetaBootstrap creates a new Bootstrap object
func NewMetaBootstrap(
	poolsHolder dataRetriever.MetaPoolsHolder,
	store dataRetriever.StorageService,
	blkc data.ChainHandler,
	rounder consensus.Rounder,
	blkExecutor process.BlockProcessor,
	waitTime time.Duration,
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	forkDetector process.ForkDetector,
	resolversFinder dataRetriever.ResolversFinder,
	shardCoordinator sharding.Coordinator,
	accounts state.AccountsAdapter,
	blackListHandler process.BlackListHandler,
	networkWatcher process.NetworkConnectionWatcher,
	bootStorer process.BootStorer,
	storageBootstrapper process.BootstrapperFromStorage,
	requestedItemsHandler dataRetriever.RequestedItemsHandler,
) (*MetaBootstrap, error) {

	if check.IfNil(poolsHolder) {
		return nil, process.ErrNilPoolsHolder
	}
	if check.IfNil(poolsHolder.HeadersNonces()) {
		return nil, process.ErrNilHeadersNoncesDataPool
	}
	if check.IfNil(poolsHolder.MetaBlocks()) {
		return nil, process.ErrNilMetaBlocksPool
	}

	err := checkBootstrapNilParameters(
		blkc,
		rounder,
		blkExecutor,
		hasher,
		marshalizer,
		forkDetector,
		resolversFinder,
		shardCoordinator,
		accounts,
		store,
		blackListHandler,
		networkWatcher,
		requestedItemsHandler,
	)
	if err != nil {
		return nil, err
	}

	base := &baseBootstrap{
		blkc:                  blkc,
		blkExecutor:           blkExecutor,
		store:                 store,
		headers:               poolsHolder.MetaBlocks(),
		headersNonces:         poolsHolder.HeadersNonces(),
		rounder:               rounder,
		waitTime:              waitTime,
		hasher:                hasher,
		marshalizer:           marshalizer,
		forkDetector:          forkDetector,
		shardCoordinator:      shardCoordinator,
		accounts:              accounts,
		blackListHandler:      blackListHandler,
		networkWatcher:        networkWatcher,
		bootStorer:            bootStorer,
		storageBootstrapper:   storageBootstrapper,
		requestedItemsHandler: requestedItemsHandler,
		miniBlocks:            poolsHolder.MiniBlocks(),
	}

	boot := MetaBootstrap{
		baseBootstrap: base,
	}

	base.blockBootstrapper = &boot
	base.getHeaderFromPool = boot.getMetaHeaderFromPool
	base.syncStarter = &boot
	base.requestMiniBlocks = boot.requestMiniBlocksFromHeaderWithNonceIfMissing

	//TODO: ResolversFinder should be replaced with RequestHandler after it would be refactored and RequestedItemsHandler
	//should be then removed from MetaBootstrap

	//there is one header topic so it is ok to save it
	hdrResolver, err := resolversFinder.MetaChainResolver(factory.MetachainBlocksTopic)
	if err != nil {
		return nil, err
	}

	//sync should request the missing block body on the intrashard topic
	miniBlocksResolver, err := resolversFinder.IntraShardResolver(factory.MiniBlocksTopic)
	if err != nil {
		return nil, err
	}

	//placed in struct fields for performance reasons
	base.headerStore = boot.store.GetStorer(dataRetriever.MetaBlockUnit)
	base.headerNonceHashStore = boot.store.GetStorer(dataRetriever.MetaHdrNonceHashDataUnit)

	hdrRes, ok := hdrResolver.(dataRetriever.HeaderResolver)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	base.hdrRes = hdrRes
	base.forkInfo = process.NewForkInfo()

	miniBlocksRes, ok := miniBlocksResolver.(dataRetriever.MiniBlocksResolver)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	boot.miniBlocksResolver = miniBlocksRes

	boot.chRcvHdrNonce = make(chan bool)
	boot.chRcvHdrHash = make(chan bool)
	boot.chRcvMiniBlocks = make(chan bool)

	boot.setRequestedHeaderNonce(nil)
	boot.setRequestedHeaderHash(nil)
	boot.setRequestedMiniBlocks(nil)

	boot.headersNonces.RegisterHandler(boot.receivedHeaderNonce)
	boot.headers.RegisterHandler(boot.receivedHeader)
	boot.miniBlocks.RegisterHandler(boot.receivedBodyHash)

	boot.chStopSync = make(chan bool)

	boot.statusHandler = statusHandler.NewNilStatusHandler()

	boot.syncStateListeners = make([]func(bool), 0)
	boot.requestedHashes = process.RequiredDataPool{}

	//TODO: This should be injected when BlockProcessor will be refactored
	boot.uint64Converter = uint64ByteSlice.NewBigEndianConverter()

	return &boot, nil
}

func (boot *MetaBootstrap) getBlockBody(headerHandler data.HeaderHandler) (data.BodyHandler, error) {
	header, ok := headerHandler.(*block.MetaBlock)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	hashes := make([][]byte, len(header.MiniBlockHeaders))
	for i := 0; i < len(header.MiniBlockHeaders); i++ {
		hashes[i] = header.MiniBlockHeaders[i].Hash
	}

	miniBlocks, missingMiniBlocksHashes := boot.miniBlocksResolver.GetMiniBlocks(hashes)
	if len(missingMiniBlocksHashes) > 0 {
		return nil, process.ErrMissingBody
	}

	return block.Body(miniBlocks), nil
}

func (boot *MetaBootstrap) receivedHeader(headerHash []byte) {
	header, err := process.GetMetaHeaderFromPool(headerHash, boot.headers)
	if err != nil {
		log.Trace("GetMetaHeaderFromPool", "error", err.Error())
		return
	}

	boot.processReceivedHeader(header, headerHash)
}

// StartSync method will start SyncBlocks as a go routine
func (boot *MetaBootstrap) StartSync() {
	// when a node starts it first tries to bootstrap from storage, if there already exist a database saved
	errNotCritical := boot.storageBootstrapper.LoadFromStorage()
	if errNotCritical != nil {
		log.Debug("syncFromStorer", "error", errNotCritical.Error())
	} else {
		_, numHdrs := updateMetricsFromStorage(boot.store, boot.uint64Converter, boot.marshalizer, boot.statusHandler, boot.storageBootstrapper.GetHighestBlockNonce())
		boot.blkExecutor.SetNumProcessedObj(numHdrs)
	}

	go boot.syncBlocks()
}

// SyncBlock method actually does the synchronization. It requests the next block header from the pool
// and if it is not found there it will be requested from the network. After the header is received,
// it requests the block body in the same way(pool and than, if it is not found in the pool, from network).
// If either header and body are received the ProcessBlock and CommitBlock method will be called successively.
// These methods will execute the block and its transactions. Finally if everything works, the block will be committed
// in the blockchain, and all this mechanism will be reiterated for the next block.
func (boot *MetaBootstrap) SyncBlock() error {
	return boot.syncBlock()
}

// requestHeaderWithNonce method requests a block header from network when it is not found in the pool
func (boot *MetaBootstrap) requestHeaderWithNonce(nonce uint64) {
	boot.setRequestedHeaderNonce(&nonce)
	err := boot.hdrRes.RequestDataFromNonce(nonce)
	if err != nil {
		log.Debug("RequestDataFromNonce", "error", err.Error())
		return
	}

	boot.requestedItemsHandler.Sweep()

	key := fmt.Sprintf("%d-%d", boot.shardCoordinator.SelfId(), nonce)
	err = boot.requestedItemsHandler.Add(key)
	if err != nil {
		log.Trace("add requested item with error", "error", err.Error())
	}

	log.Debug("requested header from network",
		"nonce", nonce,
	)
	log.Debug("probable highest nonce",
		"nonce", boot.forkDetector.ProbableHighestNonce(),
	)
}

// requestHeaderWithHash method requests a block header from network when it is not found in the pool
func (boot *MetaBootstrap) requestHeaderWithHash(hash []byte) {
	boot.setRequestedHeaderHash(hash)
	err := boot.hdrRes.RequestDataFromHash(hash)
	if err != nil {
		log.Debug("RequestDataFromHash", "error", err.Error())
		return
	}

	boot.requestedItemsHandler.Sweep()

	err = boot.requestedItemsHandler.Add(string(hash))
	if err != nil {
		log.Trace("add requested item with error", "error", err.Error())
	}

	log.Debug("requested header from network",
		"hash", hash,
	)
}

// getHeaderWithNonceRequestingIfMissing method gets the header with a given nonce from pool. If it is not found there, it will
// be requested from network
func (boot *MetaBootstrap) getHeaderWithNonceRequestingIfMissing(nonce uint64) (data.HeaderHandler, error) {
	hdr, _, err := process.GetMetaHeaderFromPoolWithNonce(
		nonce,
		boot.headers,
		boot.headersNonces)
	if err != nil {
		_ = process.EmptyChannel(boot.chRcvHdrNonce)
		boot.requestHeaderWithNonce(nonce)
		err := boot.waitForHeaderNonce()
		if err != nil {
			return nil, err
		}

		hdr, _, err = process.GetMetaHeaderFromPoolWithNonce(
			nonce,
			boot.headers,
			boot.headersNonces)
		if err != nil {
			return nil, err
		}
	}

	return hdr, nil
}

// getHeaderWithHashRequestingIfMissing method gets the header with a given hash from pool. If it is not found there,
// it will be requested from network
func (boot *MetaBootstrap) getHeaderWithHashRequestingIfMissing(hash []byte) (data.HeaderHandler, error) {
	hdr, err := process.GetMetaHeader(hash, boot.headers, boot.marshalizer, boot.store)
	if err != nil {
		_ = process.EmptyChannel(boot.chRcvHdrHash)
		boot.requestHeaderWithHash(hash)
		err := boot.waitForHeaderHash()
		if err != nil {
			return nil, err
		}

		hdr, err = process.GetMetaHeaderFromPool(hash, boot.headers)
		if err != nil {
			return nil, err
		}
	}

	return hdr, nil
}

func (boot *MetaBootstrap) getPrevHeader(
	header data.HeaderHandler,
	headerStore storage.Storer,
) (data.HeaderHandler, error) {

	prevHash := header.GetPrevHash()
	buffHeader, err := headerStore.Get(prevHash)
	if err != nil {
		return nil, err
	}

	prevHeader := &block.MetaBlock{}
	err = boot.marshalizer.Unmarshal(prevHeader, buffHeader)
	if err != nil {
		return nil, err
	}

	return prevHeader, nil
}

func (boot *MetaBootstrap) getCurrHeader() (data.HeaderHandler, error) {
	blockHeader := boot.blkc.GetCurrentBlockHeader()
	if blockHeader == nil {
		return nil, process.ErrNilBlockHeader
	}

	header, ok := blockHeader.(*block.MetaBlock)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	return header, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (boot *MetaBootstrap) IsInterfaceNil() bool {
	if boot == nil {
		return true
	}
	return false
}

func (boot *MetaBootstrap) haveHeaderInPoolWithNonce(nonce uint64) bool {
	_, _, err := process.GetMetaHeaderFromPoolWithNonce(
		nonce,
		boot.headers,
		boot.headersNonces)

	return err == nil
}

func (boot *MetaBootstrap) getMetaHeaderFromPool(headerHash []byte) (data.HeaderHandler, error) {
	return process.GetMetaHeaderFromPool(headerHash, boot.headers)
}

func (boot *MetaBootstrap) getBlockBodyRequestingIfMissing(headerHandler data.HeaderHandler) (data.BodyHandler, error) {
	header, ok := headerHandler.(*block.MetaBlock)
	if !ok {
		return nil, process.ErrWrongTypeAssertion
	}

	hashes := make([][]byte, len(header.MiniBlockHeaders))
	for i := 0; i < len(header.MiniBlockHeaders); i++ {
		hashes[i] = header.MiniBlockHeaders[i].Hash
	}

	boot.setRequestedMiniBlocks(nil)

	miniBlockSlice, err := boot.getMiniBlocksRequestingIfMissing(hashes)
	if err != nil {
		return nil, err
	}

	blockBody := block.Body(miniBlockSlice)

	return blockBody, nil
}

func (boot *MetaBootstrap) requestMiniBlocksFromHeaderWithNonceIfMissing(_ uint32, nonce uint64) {
	nextBlockNonce := boot.getNonceForNextBlock()
	maxNonce := core.MinUint64(nextBlockNonce+process.MaxHeadersToRequestInAdvance-1, boot.forkDetector.ProbableHighestNonce())
	if nonce < nextBlockNonce || nonce > maxNonce {
		return
	}

	header, _, err := process.GetMetaHeaderFromPoolWithNonce(
		nonce,
		boot.headers,
		boot.headersNonces)

	if err != nil {
		log.Trace("GetMetaHeaderFromPoolWithNonce", "error", err.Error())
		return
	}

	boot.requestedItemsHandler.Sweep()

	hashes := make([][]byte, 0)
	for i := 0; i < len(header.MiniBlockHeaders); i++ {
		if boot.requestedItemsHandler.Has(string(header.MiniBlockHeaders[i].Hash)) {
			continue
		}

		hashes = append(hashes, header.MiniBlockHeaders[i].Hash)
	}

	_, missingMiniBlocksHashes := boot.miniBlocksResolver.GetMiniBlocksFromPool(hashes)
	if len(missingMiniBlocksHashes) > 0 {
		err := boot.miniBlocksResolver.RequestDataFromHashArray(missingMiniBlocksHashes)
		if err != nil {
			log.Debug("RequestDataFromHashArray", "error", err.Error())
			return
		}

		for _, hash := range missingMiniBlocksHashes {
			err = boot.requestedItemsHandler.Add(string(hash))
			if err != nil {
				log.Trace("add requested item with error", "error", err.Error())
			}
		}

		log.Trace("requested in advance mini blocks",
			"num miniblocks", len(missingMiniBlocksHashes),
			"header nonce", header.Nonce,
		)
	}
}

func (boot *MetaBootstrap) isForkTriggeredByMeta() bool {
	return boot.forkInfo.IsDetected &&
		boot.forkInfo.Nonce != math.MaxUint64 &&
		boot.forkInfo.Round != math.MaxUint64 &&
		boot.forkInfo.Hash != nil
}
