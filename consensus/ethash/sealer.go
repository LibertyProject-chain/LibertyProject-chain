// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"

	// "github.com/ethereum/go-ethereum/shared"
	"github.com/zeebo/blake3"
)

const (
	// staleThreshold is the maximum depth of the acceptable stale but valid ethash solution.
	staleThreshold = 7
)

var (
	errNoMiningWork      = errors.New("no mining work available yet")
	errInvalidSealResult = errors.New("invalid or stale proof-of-work solution")
)

// Seal implements consensus.Engine, attempting to find a nonce that satisfies
// the block's difficulty requirements.
func (ethash *Ethash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	// If we're running a fake PoW, simply return a 0 nonce immediately
	if ethash.config.PowMode == ModeFake || ethash.config.PowMode == ModeFullFake {
		header := block.Header()
		header.Nonce, header.MixDigest = types.BlockNonce{}, common.Hash{}
		select {
		case results <- block.WithSeal(header):
		default:
			ethash.config.Log.Warn("Sealing result is not read by miner", "mode", "fake", "sealhash", ethash.SealHash(block.Header()))
		}
		return nil
	}
	// If we're running a shared PoW, delegate sealing to it
	if ethash.shared != nil {
		return ethash.shared.Seal(chain, block, results, stop)
	}
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})

	ethash.lock.Lock()
	threads := ethash.threads
	if ethash.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			ethash.lock.Unlock()
			return err
		}
		ethash.rand = rand.New(rand.NewSource(seed.Int64()))
	}
	ethash.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	// Push new work to remote sealer
	if ethash.remote != nil {
		ethash.remote.workCh <- &sealTask{block: block, results: results}
	}
	var (
		pend   sync.WaitGroup
		locals = make(chan *types.Block)
	)
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			ethash.mine(block, id, nonce, abort, locals)
		}(i, uint64(ethash.rand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	go func() {
		var result *types.Block
		select {
		case <-stop:
			// Outside abort, stop all miner threads
			close(abort)
		case result = <-locals:
			// One of the threads found a block, abort all others
			select {
			case results <- result:
			default:
				ethash.config.Log.Warn("Sealing result is not read by miner", "mode", "local", "sealhash", ethash.SealHash(block.Header()))
			}
			close(abort)
		case <-ethash.update:
			// Thread count was changed on user request, restart
			close(abort)
			if err := ethash.Seal(chain, block, results, stop); err != nil {
				ethash.config.Log.Error("Failed to restart sealing after update", "err", err)
			}
		}
		// Wait for all miners to terminate and return the block
		pend.Wait()
	}()
	return nil
}

func (ethash *Ethash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	var (
		header    = block.Header()
		target    = new(big.Int).Div(two256, header.Difficulty)
		attempts  = int64(0)
		nonce     = seed
		powBuffer = new(big.Int)
		iterCount = 312688 // Number of hashing iterations
	)
	logger := ethash.config.Log.New("miner", id)
	logger.Trace("Started Blake3 search for new nonces", "seed", seed)

search:
	for {
		select {
		case <-abort:
			logger.Trace("Blake3 nonce search aborted", "attempts", nonce-seed)
			ethash.hashrate.Mark(attempts)
			break search
		default:
			attempts++
			if (attempts % (1 << 15)) == 0 {
				ethash.hashrate.Mark(attempts)
				attempts = 0
			}

			// Get SealHash (header without Nonce and MixDigest)
			sealHash := ethash.SealHash(header).Bytes()

			// Create buffer for storage SealHash + Nonce
			var buffer bytes.Buffer
			buffer.Write(sealHash)
			nonceBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(nonceBytes, nonce)
			buffer.Write(nonceBytes)

			// Perform multiple hashing
			hashResult := blake3.Sum256(buffer.Bytes())
			for i := 0; i < iterCount; i++ {
				hashResult = blake3.Sum256(hashResult[:])
			}

			// Convert hash to a number for comparison with target
			powBuffer.SetBytes(hashResult[:])
			if powBuffer.Cmp(target) <= 0 {

				header = types.CopyHeader(header)
				header.Nonce = types.EncodeNonce(nonce)
				header.MixDigest = common.BytesToHash(hashResult[:])

				logger.Debug("Found valid block", "nonce", nonce, "mixDigest", hex.EncodeToString(hashResult[:]), "sealHash", hex.EncodeToString(sealHash))

				select {
				case found <- block.WithSeal(header):
					logger.Trace("Blake3 nonce found and reported", "attempts", nonce-seed, "nonce", nonce)
				case <-abort:
					logger.Trace("Blake3 nonce found but discarded", "attempts", nonce-seed, "nonce", nonce)
				}
				break search
			}
			nonce++
		}
	}
}

// This is the timeout for HTTP requests to notify external miners.
const remoteSealerTimeout = 1 * time.Second

type remoteSealer struct {
	works        map[common.Hash]*types.Block
	rates        map[common.Hash]hashrate
	currentBlock *types.Block
	currentWork  [4]string
	notifyCtx    context.Context
	cancelNotify context.CancelFunc // cancels all notification requests
	reqWG        sync.WaitGroup     // tracks notification request goroutines

	ethash       *Ethash
	noverify     bool
	notifyURLs   []string
	results      chan<- *types.Block
	workCh       chan *sealTask   // Notification channel to push new work and relative result channel to remote sealer
	fetchWorkCh  chan *sealWork   // Channel used for remote sealer to fetch mining work
	submitWorkCh chan *mineResult // Channel used for remote sealer to submit their mining result
	fetchRateCh  chan chan uint64 // Channel used to gather submitted hash rate for local or remote sealer.
	submitRateCh chan *hashrate   // Channel used for remote sealer to submit their mining hashrate
	requestExit  chan struct{}
	exitCh       chan struct{}
}

// sealTask wraps a seal block with relative result channel for remote sealer thread.
type sealTask struct {
	block   *types.Block
	results chan<- *types.Block
}

// mineResult wraps the pow solution parameters for the specified block.
type mineResult struct {
	nonce        types.BlockNonce
	mixDigest    common.Hash
	hash         common.Hash
	errc         chan error
	minerAddress common.Address // Добавляем поле для хранения адреса майнера
}

// hashrate wraps the hash rate submitted by the remote sealer.
type hashrate struct {
	id   common.Hash
	ping time.Time
	rate uint64

	done chan struct{}
}

// sealWork wraps a seal work package for remote sealer.
type sealWork struct {
	errc chan error
	res  chan [4]string
}

func startRemoteSealer(ethash *Ethash, urls []string, noverify bool) *remoteSealer {
	ctx, cancel := context.WithCancel(context.Background())
	s := &remoteSealer{
		ethash:       ethash,
		noverify:     noverify,
		notifyURLs:   urls,
		notifyCtx:    ctx,
		cancelNotify: cancel,
		works:        make(map[common.Hash]*types.Block),
		rates:        make(map[common.Hash]hashrate),
		workCh:       make(chan *sealTask),
		fetchWorkCh:  make(chan *sealWork),
		submitWorkCh: make(chan *mineResult),
		fetchRateCh:  make(chan chan uint64),
		submitRateCh: make(chan *hashrate),
		requestExit:  make(chan struct{}),
		exitCh:       make(chan struct{}),
	}
	go s.loop()
	return s
}

func (s *remoteSealer) loop() {
	defer func() {
		s.ethash.config.Log.Trace("Ethash remote sealer is exiting")
		s.cancelNotify()
		s.reqWG.Wait()
		close(s.exitCh)
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case work := <-s.workCh:
			// Logging receipt of new mining task
			// log.Printf("Liberty Project: New mining work received - BlockNumber=%d, BlockHash=%s", work.block.NumberU64(), work.block.Hash().Hex())

			// Update current work with new block
			s.results = work.results
			s.makeWork(work.block)
			s.notifyWork()
			// log.Printf("Liberty Project: Current work updated - BlockNumber=%d", s.currentBlock.NumberU64())

		case work := <-s.fetchWorkCh:
			// Logging request to get current work
			// log.Printf("Liberty Project: Fetching current mining work")

			// Return current work to remote miner
			if s.currentBlock == nil {
				work.errc <- errNoMiningWork
				// log.Printf("Liberty Project: No current mining work available")
			} else {
				work.res <- s.currentWork
				// log.Printf("Liberty Project: Provided current work - BlockNumber=%d", s.currentBlock.NumberU64())
			}

		case result := <-s.submitWorkCh:
			// log.Printf("Liberty Projec: Received submitted work - MinerAddress=%s, Nonce=%d, Hash=%s", result.minerAddress.Hex(), result.nonce, result.hash.Hex())

			if s.submitWork(result.nonce, result.mixDigest, result.hash, result.minerAddress) {
				result.errc <- nil
				// log.Printf("Liberty Project: Work submission successful - BlockNumber=%d, SealHash=%s", s.currentBlock.NumberU64(), result.hash.Hex())
			} else {
				result.errc <- errInvalidSealResult
				// log.Printf("Liberty Project: Work submission failed - BlockNumber=%d, SealHash=%s", s.currentBlock.NumberU64(), result.hash.Hex())
			}

		case result := <-s.submitRateCh:
			// log.Printf("Liberty Project: Hash rate submission received - ID=%s, Rate=%d", result.id, result.rate)
			s.rates[result.id] = hashrate{rate: result.rate, ping: time.Now()}
			close(result.done)

		case req := <-s.fetchRateCh:
			// Logging total hash rate retrieval
			var total uint64
			for _, rate := range s.rates {
				total += rate.rate
			}
			// log.Printf("Liberty Project: Total hash rate fetched - TotalRate=%d", total)
			req <- total

		case <-ticker.C:
			// Clean up outdated hash rate data
			for id, rate := range s.rates {
				if time.Since(rate.ping) > 10*time.Second {
					delete(s.rates, id)
					// log.Printf("Liberty Project: Stale rate cleared - RateID=%s", id)
				}
			}
			// Clean up outdated blocks from queue
			if s.currentBlock != nil {
				for hash, block := range s.works {
					if block.NumberU64()+staleThreshold <= s.currentBlock.NumberU64() {
						delete(s.works, hash)
						// log.Printf("Liberty Project: Stale block cleared - BlockNumber=%d, BlockHash=%s", block.NumberU64(), hash.Hex())
					}
				}
			}

		case <-s.requestExit:
			// log.Printf("Liberty Project: Remote sealer exiting")
			return
		}
	}
}

// makeWork creates a work package for external miner.
//
// The work package consists of 3 strings:
//
//	result[0], 32 bytes hex encoded current block header pow-hash
//	result[1], 32 bytes hex encoded seed hash used for DAG
//	result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//	result[3], hex encoded block number
func (s *remoteSealer) makeWork(block *types.Block) {
	hash := s.ethash.SealHash(block.Header())
	s.currentWork[0] = hash.Hex()
	s.currentWork[1] = common.BytesToHash(SeedHash(block.NumberU64())).Hex()
	s.currentWork[2] = common.BytesToHash(new(big.Int).Div(two256, block.Difficulty()).Bytes()).Hex()
	s.currentWork[3] = hexutil.EncodeBig(block.Number())

	// Trace the seal work fetched by remote sealer.
	s.currentBlock = block
	s.works[hash] = block
}

// notifyWork notifies all the specified mining endpoints of the availability of
// new work to be processed.
func (s *remoteSealer) notifyWork() {
	work := s.currentWork

	// Encode the JSON payload of the notification. When NotifyFull is set,
	// this is the complete block header, otherwise it is a JSON array.
	var blob []byte
	if s.ethash.config.NotifyFull {
		blob, _ = json.Marshal(s.currentBlock.Header())
	} else {
		blob, _ = json.Marshal(work)
	}

	s.reqWG.Add(len(s.notifyURLs))
	for _, url := range s.notifyURLs {
		go s.sendNotification(s.notifyCtx, url, blob, work)
	}
}

func (s *remoteSealer) sendNotification(ctx context.Context, url string, json []byte, work [4]string) {
	defer s.reqWG.Done()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(json))
	if err != nil {
		s.ethash.config.Log.Warn("Can't create remote miner notification", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, remoteSealerTimeout)
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.ethash.config.Log.Warn("Failed to notify remote miner", "err", err)
	} else {
		s.ethash.config.Log.Trace("Notified remote miner", "miner", url, "hash", work[0], "target", work[2])
		resp.Body.Close()
	}
}

// logWithTimestamp logs a message with timestamp in a consistent format
func logWithTimestamp(level string, message string, fields map[string]interface{}) {
	timestamp := time.Now().Format("01-02|15:04:05.000")
	fieldStrings := []string{}
	for key, value := range fields {
		fieldStrings = append(fieldStrings, fmt.Sprintf("%s=%v", key, value))
	}
	fmt.Printf("%s [%s] %s: %s %s\n", level, timestamp, "Liberty-Node", message, strings.Join(fieldStrings, " "))
}

func (s *remoteSealer) submitWork(nonce types.BlockNonce, mixDigest common.Hash, sealhash common.Hash, minerAddress common.Address) bool {
	// Check if work with this sealhash exists
	block := s.works[sealhash]
	if block == nil {
		logWithTimestamp("WARN", "Work submitted but none pending", map[string]interface{}{
			"sealhash":  sealhash.Hex(),
			"curnumber": s.currentBlock.NumberU64(),
		})
		return false
	}

	// Prepare header with received nonce and mixDigest
	header := block.Header()
	header.Nonce = nonce
	header.MixDigest = mixDigest
	logWithTimestamp("INFO", "Prepared header", map[string]interface{}{
		"nonce":     fmt.Sprintf("%x", nonce),
		"mixDigest": mixDigest.Hex(),
		"sealhash":  sealhash.Hex(),
	})

	start := time.Now()
	if !s.noverify {
		// Verify PoW with original coinbase, verification by externalMinerAddress
		if err := s.ethash.verifySeal(nil, header, true); err != nil {
			logWithTimestamp("WARN", "Invalid proof-of-work submitted", map[string]interface{}{
				"sealhash": sealhash.Hex(),
				"elapsed":  common.PrettyDuration(time.Since(start)),
				"err":      err.Error(),
			})
			return false
		}
		logWithTimestamp("INFO", "Proof-of-work verification succeeded", map[string]interface{}{
			"sealhash": sealhash.Hex(),
		})
	}

	// Solution is valid, prepare block for further processing
	solution := block.WithSeal(header)

	// Check for stale solution
	if solution.NumberU64()+staleThreshold > s.currentBlock.NumberU64() {
		select {
		case s.results <- solution:
			logWithTimestamp("INFO", "Work accepted", map[string]interface{}{
				"BlockNumber": solution.NumberU64(),
				"SealHash":    sealhash.Hex(),
			})
			return true
		default:
			logWithTimestamp("WARN", "Sealing result not read by miner", map[string]interface{}{
				"sealhash": sealhash.Hex(),
			})
			return false
		}
	}

	// If block is too old, log and reject
	logWithTimestamp("WARN", "Work too old", map[string]interface{}{
		"BlockNumber": solution.NumberU64(),
		"SealHash":    sealhash.Hex(),
	})
	return false
}
