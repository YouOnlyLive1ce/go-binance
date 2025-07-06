package binance

import (
	"net/http"
	"net/url"
	"time"
	"fmt"
	"github.com/gorilla/websocket"
)

// WsHandler handle raw websocket message
type WsHandler func(message []byte)

// ErrHandler handles errors
type ErrHandler func(err error)

// WsConfig webservice configuration
type WsConfig struct {
	Endpoint string
	Proxy    *string
}

func newWsConfig(endpoint string) *WsConfig {
	return &WsConfig{
		Endpoint: endpoint,
		Proxy:    getWsProxyUrl(),
	}
}

var wsServe = func(cfg *WsConfig, handler WsHandler, errHandler ErrHandler) (doneC, stopC chan struct{}, err error) {
	fmt.Println("wsServe")
	proxy := http.ProxyFromEnvironment
	if cfg.Proxy != nil {
		u, err := url.Parse(*cfg.Proxy)
		if err != nil {
			return nil, nil, err
		}
		proxy = http.ProxyURL(u)
	}
	Dialer := websocket.Dialer{
		Proxy:             proxy,
		HandshakeTimeout:  45 * time.Second,
		EnableCompression: true,
	}

	c, _, err := Dialer.Dial(cfg.Endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	c.SetReadLimit(655350)      //connection
	doneC = make(chan struct{}) //error from websocket.Conn.ReadMessage
	stopC = make(chan struct{}) //closed by the client.
	go func() {
		defer close(doneC)
		if WebsocketKeepalive {
			// This function overwrites the default ping frame handler
			// sent by the websocket API server
			WebsocketTimeout2:=time.Second * 30
			keepAlive(c, WebsocketTimeout2)
		}

		// Wait for the stopC channel to be closed.  We do that in a
		// separate goroutine because ReadMessage is a blocking
		// operation.
		silent := false
		go func() {
			select {
			case <-stopC:
				silent = true
			case <-doneC:
			}
			c.Close()
		}()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if !silent {
					errHandler(err)
				}
				return
			}
			handler(message)
		}
	}()
	return
}

func keepAlive(c *websocket.Conn, timeout time.Duration) {
	fmt.Println("PING PONG")
	ticker := time.NewTicker(timeout)

	lastResponse := time.Now()

	c.SetPingHandler(func(pingData string) error {
		// Respond with Pong using the server's PING payload
		err := c.WriteControl(
			websocket.PongMessage,
			[]byte(pingData),
			time.Now().Add(WebsocketPongTimeout), // Short deadline to ensure timely response
		)
		if err != nil {
			return err
		}
		fmt.Println("Pong scheduled")
		lastResponse = time.Now()

		return nil
	})
	fmt.Println("goroutine")
	go func() {
		defer ticker.Stop()
		for {
			<-ticker.C
			if time.Since(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}

var WsGetReadWriteConnection = func(cfg *WsConfig) (*websocket.Conn, error) {
	proxy := http.ProxyFromEnvironment
	if cfg.Proxy != nil {
		u, err := url.Parse(*cfg.Proxy)
		if err != nil {
			return nil, err
		}
		proxy = http.ProxyURL(u)
	}

	Dialer := websocket.Dialer{
		Proxy:             proxy,
		HandshakeTimeout:  45 * time.Second,
		EnableCompression: false,
	}

	c, _, err := Dialer.Dial(cfg.Endpoint, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}
