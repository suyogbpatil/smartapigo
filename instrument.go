package smartapigo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Instrument struct {
	Token          string `json:"token"`
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	Expiry         string `json:"expiry"`
	Strike         string `json:"strike"`
	LotSize        string `json:"lotsize"`
	InstrumentType string `json:"instrumenttype"`
	ExchSeg        string `json:"exch_seg"`
	TickSize       string `json:"tick_size"`
}

var Instruments []Instrument

func (c *Client) DownloadSymbols() error {

	now := time.Now()
	checkTime := time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 00, 00, now.Location())
	fileInfo, err := os.Stat("instruments.json")
	if err == nil {
		downloadTimeDiff := fileInfo.ModTime().Sub(checkTime).Hours()
		log.Printf("File already Downlaoded  %v before : ", downloadTimeDiff)
		if downloadTimeDiff < 24 && downloadTimeDiff > 0 {
			return nil
		}

	}

	log.Println("downdloading instrumetns to file", err)

	request, err := http.NewRequest(http.MethodGet, URIGetIntruments, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("json error : %v", err.Error())
	}

	err = json.Unmarshal(resBody, &Instruments)
	if err != nil {
		return fmt.Errorf("json error : %v", err.Error())
	}

	file, err := json.MarshalIndent(Instruments, "", " ")
	if err != nil {
		return fmt.Errorf("file writing error : %v", err.Error())
	}
	_ = os.WriteFile("instruments.json", file, 0644)

	return nil
}

func FindInstrument(symbol, exchange string) (*Instrument, error) {

	jsonFile, err := os.Open("instruments.json")
	if err != nil {
		return nil, err
	}

	bytedata, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytedata, &Instruments)
	if err != nil {
		return nil, err
	}

	for _, i := range Instruments {
		if i.Symbol == symbol && i.ExchSeg == exchange {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("instrument not found")
}
