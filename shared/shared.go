package shared

import (
	"log"
	"math/big"
	"strings"

	// "sync"

	"github.com/ethereum/go-ethereum/common"
)

// Epoch structure for representing an epoch
type Epoch struct {
	Name          string
	StartBlock    uint64
	TotalReward   *big.Int
	MinerReward   *big.Int
	StakingReward *big.Int
	DevFund       *big.Int
	MinerAddress  *common.Address
}

// Convert coins amount to Wei
func coinsToWei(coins string) *big.Int {
	// Split string into integer and fractional parts
	parts := strings.Split(coins, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}
	// Add zeros to fractional part up to 18 digits (Wei precision)
	fracPart = fracPart + strings.Repeat("0", 18-len(fracPart))
	if len(fracPart) > 18 {
		fracPart = fracPart[:18]
	}
	// Combine integer and fractional parts
	weiStr := intPart + fracPart
	weiAmount := new(big.Int)
	weiAmount.SetString(weiStr, 10)
	return weiAmount
}

// Definition of all epochs according to the table with reduced intervals for testing
var epochs = []Epoch{
	{
		Name:          "Freedom",
		StartBlock:    1,
		TotalReward:   coinsToWei("9.0"),
		MinerReward:   coinsToWei("7.5"),
		StakingReward: coinsToWei("0.0"),
		DevFund:       coinsToWei("1.5"),
	},
	{
		Name:          "Unity",
		StartBlock:    51,
		TotalReward:   coinsToWei("9.0"),
		MinerReward:   coinsToWei("5.8"),
		StakingReward: coinsToWei("1.7"),
		DevFund:       coinsToWei("1.5"),
	},
	{
		Name:          "Justice",
		StartBlock:    101,
		TotalReward:   coinsToWei("8.5"),
		MinerReward:   coinsToWei("5.1"),
		StakingReward: coinsToWei("2.2"),
		DevFund:       coinsToWei("1.2"),
	},
	{
		Name:          "Equality",
		StartBlock:    151,
		TotalReward:   coinsToWei("8.0"),
		MinerReward:   coinsToWei("4.2"),
		StakingReward: coinsToWei("2.8"),
		DevFund:       coinsToWei("1.0"),
	},
	{
		Name:          "Prosperity",
		StartBlock:    201,
		TotalReward:   coinsToWei("7.5"),
		MinerReward:   coinsToWei("3.0"),
		StakingReward: coinsToWei("3.6"),
		DevFund:       coinsToWei("0.9"),
	},
	{
		Name:          "Integrity",
		StartBlock:    251,
		TotalReward:   coinsToWei("7.0"),
		MinerReward:   coinsToWei("3.2"),
		StakingReward: coinsToWei("3.0"),
		DevFund:       coinsToWei("0.75"),
	},
	{
		Name:          "Valor",
		StartBlock:    301,
		TotalReward:   coinsToWei("6.5"),
		MinerReward:   coinsToWei("3.4"),
		StakingReward: coinsToWei("2.5"),
		DevFund:       coinsToWei("0.6"),
	},
	{
		Name:          "Wisdom",
		StartBlock:    351,
		TotalReward:   coinsToWei("6.0"),
		MinerReward:   coinsToWei("3.7"),
		StakingReward: coinsToWei("1.8"),
		DevFund:       coinsToWei("0.5"),
	},
	{
		Name:          "Peace",
		StartBlock:    401,
		TotalReward:   coinsToWei("5.5"),
		MinerReward:   coinsToWei("3.8"),
		StakingReward: coinsToWei("1.3"),
		DevFund:       coinsToWei("0.4"),
	},
	{
		Name:          "Legacy",
		StartBlock:    451,
		TotalReward:   coinsToWei("5.0"),
		MinerReward:   coinsToWei("3.6"),
		StakingReward: coinsToWei("1.0"),
		DevFund:       coinsToWei("0.4"),
	},
	{
		Name:          "Finality",
		StartBlock:    501,
		TotalReward:   coinsToWei("4.5"),
		MinerReward:   coinsToWei("3.3"),
		StakingReward: coinsToWei("0.8"),
		DevFund:       coinsToWei("0.4"),
	},
}

// Get current epoch by block number
func GetCurrentEpoch(blockNumber uint64) *Epoch {
	if len(epochs) == 0 {
		log.Fatalf("Epochs table is empty")
	}

	for i := len(epochs) - 1; i >= 0; i-- {
		if blockNumber >= epochs[i].StartBlock {
			return &epochs[i]
		}
	}
	return &epochs[0] // Return first epoch by default
}
