package constant

// error constants
const (
	Banner                     = "QuantBot"
	Version                    = "0.0.1"
	ErrAuthorizationError      = "Authorization Error"
	ErrInsufficientPermissions = "Insufficient Permissions"
)

// exchange types
const (
	Okex       = "okex"
	Huobi      = "huobi"
	Binance    = "binance"
	GateIo     = "gateio"
	Poloniex   = "poloniex"
	OkexFuture = "Okex.future"
)

// log types
const (
	ERROR      = "ERROR"
	INFO       = "INFO"
	PROFIT     = "PROFIT"
	BUY        = "BUY"
	SELL       = "SELL"
	LONG       = "LONG"
	SHORT      = "SHORT"
	LONGCLOSE  = "LONG_CLOSE"
	SHORTCLOSE = "SHORT_CLOSE"
	CANCEL     = "CANCEL"
)

// trade types
const (
	TradeTypeBuy        = "BUY"
	TradeTypeSell       = "SELL"
	TradeTypeLong       = "LONG"
	TradeTypeShort      = "SHORT"
	TradeTypeLongClose  = "LONG_CLOSE"
	TradeTypeShortClose = "SHORT_CLOSE"
)

// some variables
var (
	Consts        = []string{"M", "M5", "M15", "M30", "H", "D", "W"}
	ExchangeTypes = []string{Okex, Huobi, Binance, GateIo, Poloniex, OkexFuture}
)
