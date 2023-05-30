package websocketstream2

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var tickerURL = url.URL{Scheme: "ws", Host: "smartapisocket.angelone.in", Path: "/smart-stream"}

type SocketClient struct {
	ClientID            string          // Client ID
	ApiKey              string          // Api Key
	AccessToken         string          // AccessToken
	FeedToken           string          // FeedToken
	Conn                *websocket.Conn // WebSocket Connection
	DataPipe            chan Quote      // Channel for quotes
	SubList             WSRequest       // Token Subscription List
	PingDuration        time.Duration   // Next Ping Time to keep connections alive default 30sec
	Reconnect           bool            // Reconnect Boolean
	MaxReconnectRetries int64           // Max Reconnect Attempts
	ReconnectDelay      time.Duration   // Min Delay in Reconnect
	ReconnectCount      int64           // Reconnect Count
	NextReconnectTime   time.Time       // Next Reconnect Time
	NextPingTime        time.Time       // Next Ping Time
}

type WSRequest struct {
	Action int    `json:"action"`
	Params Params `json:"params"`
}

type Params struct {
	Mode      int         `json:"mode"`
	TokenList []TokenList `json:"tokenList"`
}

type TokenList struct {
	ExchangeType int      `json:"exchangeType"`
	Tokens       []string `json:"tokens"`
}

// Exchange Type in Token List
const (
	NSE   = 1
	NFO   = 2
	BSE   = 3
	BSEFO = 4
	MCXFO = 5
	NCXFO = 7
	CDEFO = 13
)

type Quote struct {
	SubscriptionMode    int8
	ExchangeType        int8
	Token               string
	SequenceNumber      int64
	ExchangeTimestamp   int64
	LastTradedPrice     float64 // end of ltppacket
	LastTradedQuantity  int64
	AverageTradedPrice  float64
	VolumeTraded        int64
	TotalBuyQuantity    int64
	TotalSellQuantity   int64
	OpenPrice           float64
	HighPrice           float64
	LowPrice            float64
	ClosePrice          float64 // end of quote packet
	LastTradedTimestamp int64
	OpenInterest        int64
	Dummy               float64 //dummy field
	BestFiveData        BestFiveData
	UpperCircuitLimit   float64
	LowerCircuitLimit   float64
	YearHighPrice       float64
	YearLowPrice        float64
}

type BestFiveData struct {
	Buys  [5]BidAskPacket
	Sells [5]BidAskPacket
}

type BidAskPacket struct {
	Flag     int16
	Price    float64
	Quantity int64
	Order    int16
}

func NewSocketClient(ClientID, AccessToken, FeedToken string) *SocketClient {
	return &SocketClient{
		ClientID:            ClientID,
		AccessToken:         AccessToken,
		FeedToken:           FeedToken,
		DataPipe:            make(chan Quote),
		Reconnect:           true,
		ReconnectDelay:      time.Duration(5 * time.Second),
		MaxReconnectRetries: 300,
		ReconnectCount:      0,
		PingDuration:        time.Duration(30 * time.Second),
		NextReconnectTime:   time.Now(),
		NextPingTime:        time.Now().Add(30 * time.Second),
	}
}

func (sc *SocketClient) ServeSocket(wg *sync.WaitGroup) {

	log.Println("connecting to websocket")
	defer wg.Done()
	defer sc.Conn.Close()

	for {

		//Checking is Reconenct is true
		if sc == nil && !sc.Reconnect {
			log.Println("auto reconnect is false")
			continue
		}

		//checking max reconnect attemps
		if sc == nil && sc.ReconnectCount > sc.MaxReconnectRetries {
			log.Printf("max connection attempts achived %d\n", sc.ReconnectCount)
			continue
		}

		if sc == nil && time.Now().After(sc.NextReconnectTime) {
			var headers = http.Header{}
			headers.Add("Authorization", sc.AccessToken)
			headers.Add("x-api-key", sc.ApiKey)
			headers.Add("x-client-code", sc.ClientID)
			headers.Add("x-feed-token", sc.FeedToken)

			conn, resp, err := websocket.DefaultDialer.Dial(tickerURL.String(), headers)
			if err != nil {
				log.Println("unable to connect to websocket : ", err.Error())
				sc.NextReconnectTime = time.Now().Add(time.Duration(sc.ReconnectCount) * 2 * time.Second)
				continue
			}

			if resp.StatusCode == 401 {
				log.Println("Unable to Connect : ", resp.Header.Get("x-error-message"))
				sc.NextReconnectTime = time.Now().Add(time.Duration(sc.ReconnectCount) * 2 * time.Second)
				continue
			}

			sc.Conn = conn
			sc.ReconnectCount++
			sc.WriteMessage(sc.SubList)

		}

		if sc.Conn != nil {
			log.Println("connected to websocket")

			sc.ReadMessages()

			if time.Now().After(sc.NextPingTime) {
				err := sc.Conn.WriteMessage(websocket.TextMessage, []byte("ping"))
				if err != nil {
					log.Println("ping err : ", err.Error())
				}
			}

		}

	}

}

// ReadMessages
func (sc *SocketClient) ReadMessages() {

	if sc.Conn == nil {
		log.Println("read messages : socket client cannot be nil")
		return
	}

	i, p, err := sc.Conn.ReadMessage()
	if err != nil {
		log.Println("error read message : ", err.Error())
	}

	if i == 1 {

		log.Println("text recived : ", string(p))
	}

	if i == 2 {

		quote, err := ParseBinaryQuote(p)
		if err != nil {
			log.Println("error converting data : ", err.Error())
		}

		sc.DataPipe <- quote

	}

}

func (sc *SocketClient) WriteMessage(message WSRequest) {

	if sc.Conn == nil {
		log.Println("WS write message sc cannot be nil")
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Println("error encoding json : ", err)
		return
	}
	log.Println("jsondata", data)

	err = sc.Conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Println("write message err : ", err)
	}

	log.Printf("message sent : %+v", message)
}

func ParseBinaryQuote(p []byte) (d Quote, err error) {

	d.SubscriptionMode = int8(p[0])
	d.ExchangeType = int8(p[1])
	d.Token = removeNullBytes(string(p[2:27]))
	d.SequenceNumber = int64(binary.LittleEndian.Uint64(p[27:35]))
	d.ExchangeTimestamp = int64(binary.LittleEndian.Uint64(p[35:43]))
	d.LastTradedPrice = float64(int64((binary.LittleEndian.Uint64(p[43:51])))) / 100.00
	d.LastTradedQuantity = int64(binary.LittleEndian.Uint64(p[51:59]))
	d.AverageTradedPrice = float64(int64(binary.LittleEndian.Uint64(p[59:67]))) / 100.00
	// End of LTP Packet
	if d.SubscriptionMode >= 2 {
		d.VolumeTraded = int64(binary.LittleEndian.Uint64(p[67:75]))
		d.TotalBuyQuantity = int64(math.Float64frombits(binary.LittleEndian.Uint64(p[75:83])))
		d.TotalSellQuantity = int64(math.Float64frombits(binary.LittleEndian.Uint64(p[83:91])))
		d.OpenPrice = float64(binary.LittleEndian.Uint64(p[91:99])) / 100
		d.HighPrice = float64(int64(binary.LittleEndian.Uint64(p[99:107]))) / 100.00
		d.LowPrice = float64(int64(binary.LittleEndian.Uint64(p[107:115]))) / 100.00
		d.ClosePrice = float64(binary.LittleEndian.Uint64(p[115:123])) / 100.00

	}

	if d.SubscriptionMode == 3 {
		d.LastTradedTimestamp = int64(binary.LittleEndian.Uint64(p[123:131]))
		d.OpenInterest = int64(binary.LittleEndian.Uint64(p[131:139]))
		buycount := 0
		sellcount := 0
		for i := 0; i < 10; i++ {
			packetStart := 147 + i*20
			var bidask BidAskPacket
			bidask.Flag = int16(binary.LittleEndian.Uint16(p[packetStart : packetStart+2]))
			bidask.Quantity = int64(binary.LittleEndian.Uint64(p[packetStart+2 : packetStart+10]))
			bidask.Price = float64(int64(binary.LittleEndian.Uint64(p[packetStart+10:packetStart+18]))) / 100.00
			bidask.Order = int16(binary.LittleEndian.Uint16(p[packetStart+18 : packetStart+20]))
			if i < 5 && bidask.Flag == 1 {
				d.BestFiveData.Buys[buycount] = bidask
				buycount++
			}
			if i >= 5 && bidask.Flag == 0 {
				d.BestFiveData.Sells[sellcount] = bidask
				sellcount++
			}
		}

		d.UpperCircuitLimit = float64(int64((binary.LittleEndian.Uint32(p[347:355])))) / 100.00
		d.LowerCircuitLimit = float64(int64((binary.LittleEndian.Uint32(p[355:363])))) / 100.00
		d.YearHighPrice = float64(int64(binary.LittleEndian.Uint32(p[363:371]))) / 100.00
		d.YearLowPrice = float64(int64(binary.LittleEndian.Uint32(p[371:379]))) / 100.00
	}

	return
}

func removeNullBytes(s string) string {
	return strings.TrimRight(s, "\x00")
}
