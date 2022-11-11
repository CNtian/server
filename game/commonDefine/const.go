package commonDef

import (
	"github.com/shopspring/decimal"
)

const SR = 1000

var decInt = decimal.NewFromInt(SR)

func ScoreToClient(score int64) string {
	return decimal.NewFromInt(score).Div(decInt).Truncate(3).String()
}

func Float64ToString(score float64) string {
	return decimal.NewFromFloat(score).Truncate(3).String()
}

func Float64Mul1000ToService(score float64) int64 {
	return decimal.NewFromFloat(score).Mul(decInt).IntPart()
}

func Float64ScoreToInt64(score float64) int64 {
	return decimal.NewFromFloat(score).IntPart()
}

func TextScoreToService(score string) (int64, error) {
	dec, err := decimal.NewFromString(score)
	if err != nil {
		return 0, err
	}
	return dec.Mul(decInt).IntPart(), err
}
