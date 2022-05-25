package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	encodingJson "encoding/json"
	"fmt"
	netUrl "net/url"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/miaolz123/conver"
	"github.com/phonegapX/QuantBot/constant"
	"github.com/phonegapX/QuantBot/model"
)

const (
	message = "hello world!"
	secret  = "0933e54e76b24731a2d84b6b463ec04c"
)

// OKEX the exchange struct of okex.com
type OKEX struct {
	stockTypeMap     map[string]string
	tradeTypeMap     map[string]string
	recordsPeriodMap map[string]string
	minAmountMap     map[string]float64
	records          map[string][]Record
	host             string
	logger           model.Logger
	option           Option

	limit     float64
	lastSleep int64
	lastTimes int64
}

var mgnModes map[string]bool = map[string]bool{
	"cross":    true,
	"isolated": true,
}

// NewOKEX create an exchange struct of okex.com
func NewOKEX(opt Option) Exchange {
	return &OKEX{
		stockTypeMap: map[string]string{
			"BTC/USDT":      "BTC-USDT",
			"ETH/USDT":      "ETH-USDT",
			"EOS/USDT":      "eos_usdt",
			"ONT/USDT":      "ont_usdt",
			"QTUM/USDT":     "qtum_usdt",
			"ONT/ETH":       "ont_eth",
			"BTC/USD/SWAP":  "BTC-USD-SWAP",
			"BTC/USDT/SWAP": "BTC-USDT-SWAP",
		},
		tradeTypeMap: map[string]string{
			"buy":         constant.TradeTypeBuy,
			"sell":        constant.TradeTypeSell,
			"buy_market":  constant.TradeTypeBuy,
			"sell_market": constant.TradeTypeSell,
		},
		recordsPeriodMap: map[string]string{
			"M":   "1m",
			"M5":  "5m",
			"M15": "15m",
			"M30": "30m",
			"H":   "1H",
			"H4":  "4H",
			"D":   "1D",
			"W":   "1W",
		},
		minAmountMap: map[string]float64{
			"BTC/USDT":  0.001,
			"ETH/USDT":  0.001,
			"EOS/USDT":  0.001,
			"ONT/USDT":  0.001,
			"QTUM/USDT": 0.001,
			"ONT/ETH":   0.001,
		},
		records: make(map[string][]Record),
		// host:    "https://www.okex.com/api/v1/",
		host:   "https://www.okx.com/api/v5/",
		logger: model.Logger{TraderID: opt.TraderID, ExchangeType: opt.Type},
		option: opt,

		limit:     10.0,
		lastSleep: time.Now().UnixNano(),
	}
}

// Log print something to console
func (e *OKEX) Log(msgs ...interface{}) {
	e.logger.Log(constant.INFO, "", 0.0, 0.0, msgs...)
}

// GetType get the type of this exchange
func (e *OKEX) GetType() string {
	return e.option.Type
}

// GetName get the name of this exchange
func (e *OKEX) GetName() string {
	return e.option.Name
}

// SetLimit set the limit calls amount per second of this exchange
func (e *OKEX) SetLimit(times interface{}) float64 {
	e.limit = conver.Float64Must(times)
	return e.limit
}

// AutoSleep auto sleep to achieve the limit calls amount per second of this exchange
func (e *OKEX) AutoSleep() {
	now := time.Now().UnixNano()
	interval := 1e+9/e.limit*conver.Float64Must(e.lastTimes) - conver.Float64Must(now-e.lastSleep)
	if interval > 0.0 {
		time.Sleep(time.Duration(conver.Int64Must(interval)))
	}
	e.lastTimes = 0
	e.lastSleep = now
}

// GetMinAmount get the min trade amonut of this exchange
func (e *OKEX) GetMinAmount(stock string) float64 {
	return e.minAmountMap[stock]
}

func (e *OKEX) getAuthJSON(url string, method string, body interface{}) (json *simplejson.Json, err error) {
	p, _ := netUrl.Parse(url)
	requestPath := p.Path
	if p.RawQuery != "" {
		requestPath = p.Path + "?" + p.RawQuery
	}
	apiKey := e.option.AccessKey
	passphrase := e.option.Passphrase
	// sign := HmacSha256(timestamp+method+requestPath, e.option.SecretKey)
	var timestamp string
	var signStr string
	if method == "GET" {
		timestamp, signStr = sign("GET", requestPath, "", []byte(e.option.SecretKey))
	} else {
		j, err := encodingJson.Marshal(body)
		if err != nil {
			return nil, err
		}
		signBody := string(j)
		if body == "{}" {
			signBody = ""
		}
		timestamp, signStr = sign("POST", requestPath, signBody, []byte(e.option.SecretKey))
	}

	header := map[string]string{
		"Content-Type":         "application/json",
		"OK-ACCESS-KEY":        apiKey,
		"OK-ACCESS-SIGN":       signStr,
		"OK-ACCESS-TIMESTAMP":  timestamp,
		"OK-ACCESS-PASSPHRASE": passphrase,
		"x-simulated-trading":  "1",
	}

	e.lastTimes++
	var errs error
	var resp []byte
	if method == "GET" {
		resp, errs = getWithHeader(url, header, body)
	} else if method == "POST" {
		resp, errs = postWithHeader(url, header, body)
	}
	if errs != nil {
		return nil, errs
	}

	return simplejson.NewJson(resp)
}

// GetAccount get the account detail of this exchange
func (e *OKEX) GetAccount() interface{} {
	json, err := e.getAuthJSON(e.host+"account/balance", "GET", nil)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetAccount() balance error, ", err)
		return false
	}

	json = json.Get("data").GetIndex(0)
	// if result := json.Get("data").GetIndex(0).MustBool(); !result {
	// 	err = fmt.Errorf("GetAccount() error, the error number is %v", json.Get("error_code").MustInt())
	// 	e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetAccount() error, ", err)
	// 	return false
	// }
	return map[string]float64{
		"adjEq": conver.Float64Must(json.Get("adjEq").MustString()),
		// details := conver.Float64Must(json.Get("details").MustString())
		"imr":         conver.Float64Must(json.Get("imr").MustString()),
		"isoEq":       conver.Float64Must(json.Get("isoEq").MustString()),
		"mgnRatio":    conver.Float64Must(json.Get("mgnRatio").MustString()),
		"ordFroz":     conver.Float64Must(json.Get("ordFroz").MustString()),
		"totalEq":     conver.Float64Must(json.Get("totalEq").MustString()),
		"mmr":         conver.Float64Must(json.Get("mmr").MustString()),
		"notionalUsd": conver.Float64Must(json.Get("notionalUsd").MustString()),
		"uTime":       conver.Float64Must(json.Get("uTime").MustString()),
	}
}

// 策略下单，提供止盈止损
func (e *OKEX) TradeAlgo(instId, tdMode, side, ordType, sz string, options map[string]interface{}) interface{} {
	if _, ok := e.stockTypeMap[instId]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized stockType: ", instId)
		return false
	}
	body := map[string]interface{}{
		"instId":  e.stockTypeMap[instId],
		"tdMode":  tdMode,
		"side":    side,
		"ordType": ordType,
		"sz":      sz,
	}

	for k, v := range options {
		body[k] = v
	}

	json, err := e.getAuthJSON(e.host+"trade/order-algo", "POST", body)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "trade/order-algo error, ", err)
		return false
	}

	j := json.Get("data").GetIndex(0)
	if sCode := j.Get("sCode").MustString(); sCode != "0" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "trade/order-algo error,", j.Get("sMsg").MustString())
		return false
	}
	return j.Get("algoId").MustString()
}

// Trade place an order
func (e *OKEX) Trade(tradeType string, stockType string, _price, _amount interface{}, msgs ...interface{}) interface{} {
	stockType = strings.ToUpper(stockType)
	tradeType = strings.ToUpper(tradeType)
	price := conver.Float64Must(_price)
	amount := conver.Float64Must(_amount)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized stockType: ", stockType)
		return false
	}
	switch tradeType {
	case constant.TradeTypeBuy:
		return e.buy(stockType, price, amount, msgs...)
	case constant.TradeTypeSell:
		return e.sell(stockType, price, amount, msgs...)
	default:
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Trade() error, unrecognized tradeType: ", tradeType)
		return false
	}
}

func (e *OKEX) buy(stockType string, price, amount float64, msgs ...interface{}) interface{} {
	body := map[string]string{
		"instId":  e.stockTypeMap[stockType],
		"tdMode":  "cross",
		"side":    "buy",
		"ordType": "limit",
		"posSide": "long",
		"px":      conver.StringMust(price),
		"sz":      conver.StringMust(amount),
	}
	json, err := e.getAuthJSON(e.host+"trade/order", "POST", body)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Buy() error, ", err)
		return false
	}
	if json == nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Buy() error, ", err)
		return false
	}

	j := json.Get("data").GetIndex(0)
	if sCode := j.Get("sCode").MustString(); sCode != "0" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Buy() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	e.logger.Log(constant.BUY, stockType, price, amount, msgs...)
	return j.Get("ordId").MustString()
}

func (e *OKEX) sell(stockType string, price, amount float64, msgs ...interface{}) interface{} {
	body := map[string]string{
		"instId":  e.stockTypeMap[stockType],
		"tdMode":  "cross",
		"side":    "sell",
		"ordType": "limit",
		"posSide": "short",
		"px":      conver.StringMust(price),
		"sz":      conver.StringMust(amount),
	}
	json, err := e.getAuthJSON(e.host+"trade/order", "POST", body)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Buy() error, ", err)
		return false
	}
	if json == nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Sell() error, ", err)
		return false
	}

	j := json.Get("data").GetIndex(0)
	if sCode := j.Get("sCode").MustString(); sCode != "0" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "Sell() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	e.logger.Log(constant.SELL, stockType, price, amount, msgs...)
	return j.Get("ordId").MustString()
}

// GetOrder get details of an order
func (e *OKEX) GetOrder(instId string, option ...interface{}) interface{} {
	instId = strings.ToUpper(instId)
	if _, ok := e.stockTypeMap[instId]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() error, unrecognized stockType: ", instId)
		return false
	}
	params := []string{
		"instId=" + e.stockTypeMap[instId],
	}

	query := ""
	for i, v := range option {
		var value string
		var ok bool
		if value, ok = v.(string); !ok {
			e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() params type is error: ", v)
			return false
		}
		switch i {
		case 0:
			params = append(params, "ordId="+value)
			break
		case 1:
			params = append(params, "clOrdId="+value)
			break
		}
	}

	for index, v := range params {
		if index == 0 {
			query = v
		} else {
			query = query + "&" + v
		}
	}
	json, err := e.getAuthJSON(e.host+"trade/order?"+query, "GET", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "/api/v5/trade/order error, ", err)
		return false
	}

	if code := json.Get("code").MustString(); code != "0" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "/api/v5/trade/order error ")
		return false
	}

	json = json.Get("data")
	count := len(json.MustArray())
	orders := []Order{}
	for i := 0; i < count; i++ {
		orderJSON := json.GetIndex(i)
		order := Order{
			ID:         fmt.Sprint(orderJSON.Get("ordId").MustString()),
			Price:      conver.Float64Must(orderJSON.Get("px").MustString()),
			Amount:     conver.Float64Must(orderJSON.Get("sz").MustString()),
			DealAmount: conver.Float64Must(0),
			TradeType:  e.tradeTypeMap[orderJSON.Get("side").MustString()],
			StockType:  instId,
			Pnl:        conver.Float64Must(orderJSON.Get("pnl").MustString()),
		}
		orders = append(orders, order)
	}
	return orders
}

// GetOrder get details of an order
func (e *OKEX) GetOrderHistosy(instId, instType string, option ...map[string]interface{}) interface{} {
	instId = strings.ToUpper(instId)
	if _, ok := e.stockTypeMap[instId]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() error, unrecognized stockType: ", instId)
		return false
	}
	params := []string{
		"instId=" + e.stockTypeMap[instId],
		"instType=" + instType,
	}

	for _, value := range option {
		for k, v := range value {
			var value string
			var ok bool
			if value, ok = v.(string); !ok {
				e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() params type is error: ", k)
				return false
			}
			switch k {
			case "uly":
				params = append(params, "uly="+value)
				break
			case "ordType":
				params = append(params, "ordType="+value)
				break
			case "state":
				params = append(params, "state="+value)
			case "category":
				params = append(params, "category="+value)
			case "after":
				params = append(params, "after="+value)
			case "before":
				params = append(params, "before="+value)
			case "limit":
				params = append(params, "limit="+value)
			}
		}
	}

	query := ""
	for index, v := range params {
		if index == 0 {
			query = v
		} else {
			query = query + "&" + v
		}
	}
	json, err := e.getAuthJSON(e.host+"trade/orders-history?"+query, "GET", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrder() error, ", err)
		return false
	}

	if code := json.Get("code").MustString(); code != "0" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "trade/orders-history error ")
		return false
	}

	json = json.Get("data")
	count := len(json.MustArray())
	orders := []Order{}
	for i := 0; i < count; i++ {
		orderJSON := json.GetIndex(i)
		order := Order{
			ID:         fmt.Sprint(orderJSON.Get("ordId").MustString()),
			Price:      conver.Float64Must(orderJSON.Get("px").MustString()),
			Amount:     conver.Float64Must(orderJSON.Get("sz").MustString()),
			DealAmount: conver.Float64Must(0),
			TradeType:  e.tradeTypeMap[orderJSON.Get("side").MustString()],
			StockType:  instId,
			Pnl:        conver.Float64Must(orderJSON.Get("pnl").MustString()),
		}
		orders = append(orders, order)
	}
	return orders
}

// GetOrders get all unfilled orders
func (e *OKEX) GetOrders(stockType string) interface{} {
	// stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrders() error, unrecognized stockType: ", stockType)
		return false
	}
	params := []string{
		"instId=" + e.stockTypeMap[stockType],
		"instType=SPOT",
	}

	query := ""
	for index, v := range params {
		if index == 0 {
			query = v
		} else {
			query = query + "&" + v
		}
	}
	json, err := e.getAuthJSON(e.host+"trade/orders-pending?"+query, "GET", nil)

	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetOrders() error, ", err)
		return false
	}

	orders := []Order{}
	ordersJSON := json.Get("data")
	count := len(ordersJSON.MustArray())
	for i := 0; i < count; i++ {
		orderJSON := ordersJSON.GetIndex(i)
		orders = append(orders, Order{
			ID:         fmt.Sprint(orderJSON.Get("ordId").MustString()),
			Price:      conver.Float64Must(orderJSON.Get("px").MustString()),
			Amount:     conver.Float64Must(orderJSON.Get("sz").MustString()),
			DealAmount: conver.Float64Must(orderJSON.Get("accFillSz").MustString()),
			TradeType:  e.tradeTypeMap[orderJSON.Get("ordType").MustString()],
			StockType:  stockType,
		})
	}
	return orders
}

// GetTrades get all filled orders recently
func (e *OKEX) GetTrades(stockType string) interface{} {
	stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, unrecognized stockType: ", stockType)
		return false
	}
	params := []string{
		"symbol=" + e.stockTypeMap[stockType],
		"status=1",
		"current_page=1",
		"page_length=200",
	}
	json, err := e.getAuthJSON(e.host+"order_history.do", "GET", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	orders := []Order{}
	ordersJSON := json.Get("orders")
	count := len(ordersJSON.MustArray())
	for i := 0; i < count; i++ {
		orderJSON := ordersJSON.GetIndex(i)
		orders = append(orders, Order{
			ID:         fmt.Sprint(orderJSON.Get("order_id").Interface()),
			Price:      orderJSON.Get("price").MustFloat64(),
			Amount:     orderJSON.Get("amount").MustFloat64(),
			DealAmount: orderJSON.Get("deal_amount").MustFloat64(),
			TradeType:  e.tradeTypeMap[orderJSON.Get("type").MustString()],
			StockType:  stockType,
		})
	}
	return orders
}

// CancelOrder cancel an order
func (e *OKEX) CancelOrder(order Order) bool {
	params := []string{
		"symbol=" + e.stockTypeMap[order.StockType],
		"order_id=" + order.ID,
	}
	json, err := e.getAuthJSON(e.host+"cancel_order.do", "GET", params)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "CancelOrder() error, ", err)
		return false
	}
	if result := json.Get("result").MustBool(); !result {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "CancelOrder() error, the error number is ", json.Get("error_code").MustInt())
		return false
	}
	e.logger.Log(constant.CANCEL, order.StockType, order.Price, order.Amount-order.DealAmount, order)
	return true
}

// getTicker get market ticker & depth
func (e *OKEX) getTicker(stockType string, sizes ...interface{}) (ticker Ticker, err error) {
	stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		err = fmt.Errorf("GetTicker() error, unrecognized stockType: %+v", stockType)
		return
	}
	size := 20
	if len(sizes) > 0 && conver.IntMust(sizes[0]) > 0 {
		size = conver.IntMust(sizes[0])
	}
	resp, err := get(fmt.Sprintf("%vmarket/books?instId=%v&sz=%v", e.host, e.stockTypeMap[stockType], size))
	if err != nil {
		err = fmt.Errorf("GetTicker() error, %+v", err)
		return
	}
	json, err := simplejson.NewJson(resp)
	if err != nil {
		err = fmt.Errorf("GetTicker() error, %+v", err)
		return
	}

	data := json.Get("data").GetIndex(0)
	depthsJSON := data.Get("bids")
	for i := 0; i < len(depthsJSON.MustArray()); i++ {
		depthJSON := depthsJSON.GetIndex(i)
		// d, _ := conver.Float64(depthJSON.GetIndex(0).MustString())
		// d2, _ := conver.Float64(depthJSON.GetIndex(1).MustString())
		price, amount := getPriceByJson(depthJSON)

		ticker.Bids = append(ticker.Bids, OrderBook{
			Price:  price,
			Amount: amount,
		})
	}
	depthsJSON = data.Get("asks")
	for i := len(depthsJSON.MustArray()); i > 0; i-- {
		depthJSON := depthsJSON.GetIndex(i - 1)
		price, amount := getPriceByJson(depthJSON)
		ticker.Asks = append(ticker.Asks, OrderBook{
			Price:  price,
			Amount: amount,
		})
	}
	if len(ticker.Bids) < 1 || len(ticker.Asks) < 1 {
		err = fmt.Errorf("GetTicker() error, can not get enough Bids or Asks")
		return
	}
	ticker.Buy = ticker.Bids[0].Price
	ticker.Sell = ticker.Asks[0].Price
	ticker.Mid = (ticker.Buy + ticker.Sell) / 2
	return
}

// GetTicker get market ticker & depth
func (e *OKEX) GetTicker(stockType string, sizes ...interface{}) interface{} {
	ticker, err := e.getTicker(stockType, sizes...)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, err)
		return false
	}
	return ticker
}

// GetRecords get candlestick data
func (e *OKEX) GetRecords(stockType, period string, sizes ...interface{}) interface{} {
	stockType = strings.ToUpper(stockType)
	if _, ok := e.stockTypeMap[stockType]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, unrecognized stockType: ", stockType)
		return false
	}
	if _, ok := e.recordsPeriodMap[period]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, unrecognized period: ", period)
		return false
	}
	size := 200
	if len(sizes) > 0 && conver.IntMust(sizes[0]) > 0 {
		size = conver.IntMust(sizes[0])
	}
	resp, err := get(fmt.Sprintf("%vmarket/candles?instId=%v&bar=%v&limit=%v", e.host, e.stockTypeMap[stockType], e.recordsPeriodMap[period], size))
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, ", err)
		return false
	}
	json, err := simplejson.NewJson(resp)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetRecords() error, ", err)
		return false
	}
	recordsNew := []Record{}
	json = json.Get("data")
	for i := len(json.MustArray()); i > 0; i-- {
		recordJSON := json.GetIndex(i - 1)
		recordTime := conver.Int64Must(recordJSON.GetIndex(0).MustString())
		recordsNew = append([]Record{{
			Time:   recordTime,
			Open:   conver.Float64Must(recordJSON.GetIndex(1).MustString()),
			High:   conver.Float64Must(recordJSON.GetIndex(2).MustString()),
			Low:    conver.Float64Must(recordJSON.GetIndex(3).MustString()),
			Close:  conver.Float64Must(recordJSON.GetIndex(4).MustString()),
			Volume: conver.Float64Must(recordJSON.GetIndex(5).MustString()),
		}}, recordsNew...)
	}
	e.records[period] = append(e.records[period], recordsNew...)
	if len(e.records[period]) > size {
		e.records[period] = e.records[period][len(e.records[period])-size : len(e.records[period])]
	}
	return e.records[period]
}

// GetPositions get the positions detail of this exchange
func (e *OKEX) GetPositions(options ...interface{}) interface{} {
	params := []string{}
	for index, value := range options {
		var v string
		var ok bool
		if v, ok = value.(string); !ok {
			e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetPositions() error, unrecognized stockType: ", v)
			return false
		}
		switch index {
		case 0:
			if _, ok := e.stockTypeMap[v]; !ok {
				e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetTrades() error, unrecognized instId: ", v)
				return false
			}
			params = append(params, "?instId="+e.stockTypeMap[v])
			break
		case 1:
			params = append(params, "instType="+v)
			break
		case 2:
			params = append(params, "posId="+v)
			break
		}
	}

	json, err := e.getAuthJSON(e.host+"account/positions"+strings.Join(params, "&"), "GET", nil)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "GetPositions() error, ", err)
		return false
	}
	positionsJSON := json.Get("data")
	count := len(positionsJSON.MustArray())
	positions := []Position{}
	for i := 0; i < count; i++ {
		positionJSON := positionsJSON.GetIndex(i)
		fmt.Println(positionJSON)
		positions = append(positions, Position{
			MgnMode:       positionJSON.Get("mgnMode").MustString(),
			Price:         conver.Float64Must(positionJSON.Get("avgPx").MustString()),
			Leverage:      conver.IntMust(positionJSON.Get("lever").MustString()),
			Amount:        conver.Float64Must(positionJSON.Get("pos").MustString()),
			ConfirmAmount: conver.Float64Must(positionJSON.Get("pos").MustString()),
			FrozenAmount:  0.0,
			Profit:        conver.Float64Must(positionJSON.Get("upl").MustString()),
			ContractType:  positionJSON.Get("instType").MustString(),
			TradeType:     positionJSON.Get("instType").MustString(),
			InstId:        positionJSON.Get("instId").MustString(),
			PosId:         positionJSON.Get("posId").MustString(),
			PosSide:       positionJSON.Get("posSide").MustString(),
		})
	}
	return positions
}

// GetPositions get the positions detail of this exchange
func (e *OKEX) ClosePosition(instId, mgnMode, posSide string, options ...interface{}) bool {
	if _, ok := e.stockTypeMap[instId]; !ok {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "ClosePosition() error, unrecognized stockType: ", instId)
		return false
	}

	if !mgnModes[mgnMode] {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "ClosePosition() error, unrecognized mgnMode: ", mgnMode)
		return false
	}

	if posSide == "" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "ClosePosition() error, unrecognized posSide: ", posSide)
		return false
	}

	body := map[string]interface{}{
		"instId":  e.stockTypeMap[instId],
		"mgnMode": mgnMode,
		"posSide": posSide,
	}

	for k, v := range options {
		if k == 0 {
			body["ccy"] = v
		} else if k == 1 {
			body["autoCxl"] = v
		}
	}

	json, err := e.getAuthJSON(e.host+"trade/close-position", "POST", body)
	if err != nil {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "ClosePosition() error, ", err)
		return false
	}

	if code := json.Get("code").MustString(); code != "0" {
		e.logger.Log(constant.ERROR, "", 0.0, 0.0, "trade/close-position error ")
		return false
	}

	return true
}

func sign(method, path, body string, secretKey []byte) (string, string) {
	format := "2006-01-02T15:04:05.999Z07:00"
	t := time.Now().UTC().Format(format)
	ts := fmt.Sprint(t)
	s := ts + method + path + body
	p := []byte(s)
	h := hmac.New(sha256.New, secretKey)
	h.Write(p)
	return ts, base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func getPriceByJson(json *simplejson.Json) (float64, float64) {
	var price float64
	var amount float64
	for i, s := range json.MustStringArray() {
		if i == 0 {
			price, _ = conver.Float64(s)
		}

		if i == 1 {
			amount, _ = conver.Float64(s)
		}
	}

	return price, amount
}
