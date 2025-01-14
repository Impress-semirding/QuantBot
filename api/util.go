package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	encodingJson "encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/miaolz123/conver"
)

// var client = http.DefaultClient

var client http.Client = http.Client{
	Timeout: time.Millisecond * 1000, // Set 1s timeout.
}

type timeout interface {
	Timeout() bool
}

// Position struct
type Position struct {
	InstId        string
	MgnMode       string
	Price         float64 //价格
	Leverage      int     //杠杆比例
	Amount        float64 //总合约数量
	ConfirmAmount float64
	FrozenAmount  float64 //冻结的合约数量
	Profit        float64 //收益
	ContractType  string  //合约类型
	TradeType     string  //交易类型
	StockType     string  //货币类型
	PosId         string  //持仓ID
	PosSide       string  //持仓方向,多还是空
}

// Order struct
type Order struct {
	ID         string  //订单ID
	Price      float64 //价格
	Amount     float64 //总量
	DealAmount float64 //成交量
	Fee        float64 //这个订单的交易费
	TradeType  string  //交易类型
	StockType  string  //货币类型
	Pnl        float64
}

// Record struct
type Record struct {
	Time   int64   //unix时间戳
	Open   float64 //开盘价
	High   float64 //最高价
	Low    float64 //最低价
	Close  float64 //收盘价
	Volume float64 //交易量
}

// OrderBook struct
type OrderBook struct {
	Price  float64 //价格
	Amount float64 //市场深度量
}

// Ticker struct
type Ticker struct {
	Bids []OrderBook //买单市场深度列表
	Buy  float64     //买一价, Bids[0].Price
	Mid  float64     //(Buy + Sell) / 2
	Sell float64     //卖一价, Asks[0].Price
	Asks []OrderBook //卖单市场深度列表
}

func base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func signMd5(params []string) string {
	m := md5.New()
	m.Write([]byte(strings.Join(params, "&")))
	return hex.EncodeToString(m.Sum(nil))
}

func signSha512(params []string, key string) string {
	h := hmac.New(sha512.New, []byte(key))
	h.Write([]byte(strings.Join(params, "&")))
	return hex.EncodeToString(h.Sum(nil))
}

func signSha1(params []string, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(strings.Join(params, "&")))
	return hex.EncodeToString(h.Sum(nil))
}

func signChbtc(params []string, key string) string {
	sha := sha1.New()
	sha.Write([]byte(key))
	secret := hex.EncodeToString(sha.Sum(nil))
	h := hmac.New(md5.New, []byte(secret))
	h.Write([]byte(strings.Join(params, "&")))
	return hex.EncodeToString(h.Sum(nil))
}

func post_gateio(url string, data []string, key string, sign string) (ret []byte, err error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(strings.Join(data, "&")))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("key", key)
	req.Header.Set("sign", sign)
	resp, err := client.Do(req)
	if resp == nil {
		err = fmt.Errorf("[POST %s] HTTP Error Info: %v", url, err)
	} else if resp.StatusCode == 200 {
		ret, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	} else {
		err = fmt.Errorf("[POST %s] HTTP Status: %d, Info: %v", url, resp.StatusCode, err)
	}
	return ret, err
}

func post(url string, data []string) (ret []byte, err error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(strings.Join(data, "&")))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		if t, ok := err.(timeout); ok {
			ret = nil
			err = fmt.Errorf("timeout: %t", t.Timeout())
			return ret, err
		}
	} else if resp == nil {
		err = fmt.Errorf("[POST %s] HTTP Error Info: %v", url, err)
	} else if resp.StatusCode == 200 {
		ret, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	} else {
		err = fmt.Errorf("[POST %s] HTTP Status: %d, Info: %v", url, resp.StatusCode, err)
	}
	return ret, err
}

func postWithHeader(url string, header map[string]string, data interface{}) (ret []byte, err error) {
	j, err := encodingJson.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := string(j)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		if t, ok := err.(timeout); ok {
			ret = nil
			err = fmt.Errorf("timeout: %t", t.Timeout())
			return ret, err
		}
	} else if resp == nil {
		err = fmt.Errorf("[POST %s] HTTP Error Info: %v", url, err)
	} else if resp.StatusCode == 200 {
		ret, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	} else {
		err = fmt.Errorf("[POST %s] HTTP Status: %d, Info: %v", url, resp.StatusCode, err)
	}
	return ret, err

}

func getWithHeader(url string, header map[string]string, data interface{}) (ret []byte, err error) {

	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		if t, ok := err.(timeout); ok {
			ret = nil
			err = fmt.Errorf("timeout: %t", t.Timeout())
			return ret, err
		}
	} else if resp == nil {
		err = fmt.Errorf("[GET %s] HTTP Error Info: %v", url, err)
	} else if resp.StatusCode == 200 {
		ret, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	} else {
		err = fmt.Errorf("[GET %s] HTTP Status: %d, Info: %v", url, resp.StatusCode, err)
	}
	return ret, err

}

func get(url string) (ret []byte, err error) {
	req, err := http.NewRequest("GET", url, strings.NewReader(""))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		if t, ok := err.(timeout); ok {
			ret = nil
			err = fmt.Errorf("timeout: %t", t.Timeout())
			return ret, err
		}
	} else if resp == nil {
		err = fmt.Errorf("[GET %s] HTTP Error Info: %v", url, err)
	} else if resp.StatusCode == 200 {
		ret, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	} else {
		err = fmt.Errorf("[GET %s] HTTP Status: %d, Info: %v", url, resp.StatusCode, err)
	}
	return ret, err
}

func getFloatValueFromJsonObject(json *simplejson.Json, key string) float64 {
	return conver.Float64Must(json.Get(key).MustString())
}
