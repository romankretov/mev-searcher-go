package util

import "math/big"

func ConvertUnits(raw *big.Int, decimals uint8) *big.Rat {
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	return new(big.Rat).SetFrac(raw, divisor)
}
