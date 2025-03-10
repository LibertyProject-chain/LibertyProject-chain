package shared

import (
	"github.com/ethereum/go-ethereum/common"
)

// GetDeveloperAddresses returns the list of developer addresses
func GetDeveloperAddresses() []common.Address {
	return []common.Address{
		common.HexToAddress("0x8c80A3F122Ea3e9E9b863b8535d33F3B96eE1C92"),
		common.HexToAddress("0x2357EfDB1107eA9a316A32E33321FF405Eaff788"),
		common.HexToAddress("0xE951BA28D6945AF6798D2e9463fec3e207A6CC06"),
	}
}

// GetStakingAddresses returns the list of staking addresses
func GetStakingAddresses() []common.Address {
	return []common.Address{
		common.HexToAddress("0xc92Ca93e847CD42FD824eD17e29Bc9cb96417C08"),
		common.HexToAddress("0xBE2c7A73BAE39B92320e932E12b91D8aBC28AfE3"),
	}
}
