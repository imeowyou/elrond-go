package node

import (
	"math/big"
	"net/http"
	"net/url"

	"github.com/ElrondNetwork/elrond-go/api/errors"
	"github.com/ElrondNetwork/elrond-go/core/statistics"
	"github.com/ElrondNetwork/elrond-go/node/external"
	"github.com/ElrondNetwork/elrond-go/node/heartbeat"
	"github.com/gin-gonic/gin"
)

// FacadeHandler interface defines methods that can be used from `elrondFacade` context variable
type FacadeHandler interface {
	IsNodeRunning() bool
	StartNode() error
	GetCurrentPublicKey() string
	GetHeartbeats() ([]heartbeat.PubKeyHeartbeat, error)
	TpsBenchmark() *statistics.TpsBenchmark
	StatusMetrics() external.StatusMetricsHandler
	IsInterfaceNil() bool
}

type statisticsResponse struct {
	LiveTPS               float64                   `json:"liveTPS"`
	PeakTPS               float64                   `json:"peakTPS"`
	NrOfShards            uint32                    `json:"nrOfShards"`
	NrOfNodes             uint32                    `json:"nrOfNodes"`
	BlockNumber           uint64                    `json:"blockNumber"`
	RoundNumber           uint64                    `json:"roundNumber"`
	RoundTime             uint64                    `json:"roundTime"`
	AverageBlockTxCount   *big.Int                  `json:"averageBlockTxCount"`
	LastBlockTxCount      uint32                    `json:"lastBlockTxCount"`
	TotalProcessedTxCount *big.Int                  `json:"totalProcessedTxCount"`
	ShardStatistics       []shardStatisticsResponse `json:"shardStatistics"`
}

type shardStatisticsResponse struct {
	ShardID               uint32   `json:"shardID"`
	LiveTPS               float64  `json:"liveTPS"`
	AverageTPS            *big.Int `json:"averageTPS"`
	PeakTPS               float64  `json:"peakTPS"`
	AverageBlockTxCount   uint32   `json:"averageBlockTxCount"`
	CurrentBlockNonce     uint64   `json:"currentBlockNonce"`
	LastBlockTxCount      uint32   `json:"lastBlockTxCount"`
	TotalProcessedTxCount *big.Int `json:"totalProcessedTxCount"`
}

// Routes defines node related routes
func Routes(router *gin.RouterGroup) {
	router.GET("/address", Address)
	router.GET("/heartbeatstatus", HeartbeatStatus)
	router.GET("/statistics", Statistics)
	router.GET("/status", StatusMetrics)
}

// Address returns the information about the address passed as parameter
func Address(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(FacadeHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	currentAddress := ef.GetCurrentPublicKey()
	address, err := url.Parse(currentAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrCouldNotParsePubKey.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"address": address.String()})
}

// HeartbeatStatus respond with the heartbeat status of the node
func HeartbeatStatus(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(FacadeHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	hbStatus, err := ef.GetHeartbeats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": hbStatus})
}

// Statistics returns the blockchain statistics
func Statistics(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(FacadeHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statistics": statsFromTpsBenchmark(ef.TpsBenchmark())})
}

// StatusMetrics returns the node statistics exported by an StatusMetricsHandler
func StatusMetrics(c *gin.Context) {
	ef, ok := c.MustGet("elrondFacade").(FacadeHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errors.ErrInvalidAppContext.Error()})
		return
	}

	details, err := ef.StatusMetrics().StatusMetricsMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"details": details})
}

func statsFromTpsBenchmark(tpsBenchmark *statistics.TpsBenchmark) statisticsResponse {
	sr := statisticsResponse{}
	sr.LiveTPS = tpsBenchmark.LiveTPS()
	sr.PeakTPS = tpsBenchmark.PeakTPS()
	sr.NrOfShards = tpsBenchmark.NrOfShards()
	// TODO: Should be filled
	sr.NrOfNodes = 1
	sr.RoundTime = tpsBenchmark.RoundTime()
	sr.BlockNumber = tpsBenchmark.BlockNumber()
	sr.RoundNumber = tpsBenchmark.RoundNumber()
	sr.AverageBlockTxCount = tpsBenchmark.AverageBlockTxCount()
	sr.LastBlockTxCount = tpsBenchmark.LastBlockTxCount()
	sr.TotalProcessedTxCount = tpsBenchmark.TotalProcessedTxCount()
	sr.ShardStatistics = make([]shardStatisticsResponse, tpsBenchmark.NrOfShards())

	for i := 0; i < int(tpsBenchmark.NrOfShards()); i++ {
		ss := tpsBenchmark.ShardStatistic(uint32(i))
		sr.ShardStatistics[i] = shardStatisticsResponse{
			ShardID:               ss.ShardID(),
			LiveTPS:               ss.LiveTPS(),
			PeakTPS:               ss.PeakTPS(),
			AverageTPS:            ss.AverageTPS(),
			AverageBlockTxCount:   ss.AverageBlockTxCount(),
			CurrentBlockNonce:     ss.CurrentBlockNonce(),
			LastBlockTxCount:      ss.LastBlockTxCount(),
			TotalProcessedTxCount: ss.TotalProcessedTxCount(),
		}
	}

	return sr
}
