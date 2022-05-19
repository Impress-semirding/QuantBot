package trader

import (
	"fmt"

	"github.com/markcheno/go-talib"
)

type Talib struct {
}

func (t Talib) Rsi(prices []float64, period int) interface{} {
	result := talib.Rsi(prices, period)
	fmt.Println(result)
	return result
}

func (t Talib) Ema(prices []float64, period int) interface{} {
	result := talib.Ema(prices, period)
	fmt.Println(result)
	return result
}
