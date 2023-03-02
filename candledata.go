package smartapigo

import (
	"net/http"
	"reflect"
	"time"
)

// // LTPParams represents parameters for getting LTP.
type CandleDataParams struct {
	Exchange    string `json:"exchange"`
	SymbolToken string `json:"symboltoken"`
	Interval    string `json:"interval"`
	FromDate    string `json:"fromdate"`
	ToDate      string `json:"todate"`
}

type CandleData struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `josn:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

type Candles []CandleData

// Get CandleDataFunction Returns []timestamp,[]open,[]high,[]low,[]close,[]volume
// func (c *Client) GetCandleData(candleDataParams CandleDataParams) ([]time.Time, []float64, []float64, []float64, []float64, []float64, error) {
// 	var precandles [][]interface{}
// 	var timestamp []time.Time
// 	var open []float64
// 	var high []float64
// 	var low []float64
// 	var close []float64
// 	var volume []float64

// 	params := structToMap(candleDataParams, "json")
// 	err := c.doEnvelope(http.MethodPost, URIGetCandleData, params, nil, &precandles, true)
// 	if err != nil {
// 		return timestamp, open, high, low, close, volume, err
// 	}
// 	for _, d := range precandles {

// 		tm, _ := time.Parse("2006-01-02T15:04:05-07:00", d[0].(string))
// 		timestamp = append(timestamp, tm)
// 		open = append(open, d[1].(float64))
// 		high = append(high, d[2].(float64))
// 		low = append(low, d[3].(float64))
// 		close = append(close, d[4].(float64))
// 		volume = append(volume, d[5].(float64))

// 	}
// 	return timestamp, open, high, low, close, volume, err
// }

func (c *Client) GetCandleData(candleDataParams CandleDataParams) (Candles, error) {
	var precandles [][]interface{}
	var data Candles

	params := structToMap(candleDataParams, "json")
	err := c.doEnvelope(http.MethodPost, URIGetCandleData, params, nil, &precandles, true)
	if err != nil {
		return nil, err
	}
	for _, d := range precandles {

		var candle CandleData
		tm, _ := time.Parse("2006-01-02T15:04:05-07:00", d[0].(string))
		candle.Time = tm
		candle.Open = d[1].(float64)
		candle.High = d[2].(float64)
		candle.Low = d[3].(float64)
		candle.Close = d[4].(float64)
		candle.Volume = d[5].(float64)

		data = append(data, candle)

	}
	return data, err
}

func (c Candles) getField(fieldName string) []float64 {
	var data []float64
	for _, candle := range c {
		r := reflect.ValueOf(candle)
		f := reflect.Indirect(r).FieldByName(fieldName)
		data = append(data, f.Float())
	}
	return data
}

func (c Candles) Open() []float64 {
	return c.getField("Open")
}

func (c Candles) High() []float64 {
	return c.getField("High")
}

func (c Candles) Low() []float64 {
	return c.getField("Low")
}

func (c Candles) Close() []float64 {
	return c.getField("Close")
}

func (c Candles) Volume() []float64 {
	return c.getField("Volume")
}
