package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var pool Pool

// This will let us know if the client goes away so we can remove it from the pool
func readLoop(c *Client) {
	for {
		if _, _, err := c.Conn.NextReader(); err != nil {
			//			log.Println("readLoop", err)
			if err := c.Conn.Close(); err != nil {
				log.Print(err)
			}
			pool.Unregister <- c
			break
		}
	}
}

type WebSocketService uint8

const (
	wsFull WebSocketService = iota
	wsElectrolyser
	wsFuelCell
)

type Client struct {
	ID      string // IP address and port for the registrant
	Conn    *websocket.Conn
	Service WebSocketService
	Device  string // Which electrolyser are we looking for
}

//type Message struct {
//	Type int    `json:"type"`
//	Body string `json:"body"`
//}

type WSMessageType struct {
	data    []byte
	service WebSocketService
	device  string
}

// Pool of client registrations
type Pool struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan WSMessageType
}

func (p *Pool) Init() {
	p.Clients = make(map[*Client]bool, 5)
	p.Register = make(chan *Client, 5)
	p.Unregister = make(chan *Client, 5)
	p.Broadcast = make(chan WSMessageType, 5)
}

func (pool *Pool) StartRegister() {
	for {
		select {
		case client := <-pool.Register:
			pool.Clients[client] = true
			go readLoop(client)
			//if debugOutput {
			//	log.Println("Size of Connection Pool: ", len(pool.Clients), client.ID, " added for device ", client.Device, " on service ", client.Service)
			//}
			break
		}
	}
}
func (pool *Pool) StartUnregister() {
	for {
		select {
		case client := <-pool.Unregister:
			delete(pool.Clients, client)
			//if debugOutput {
			//	log.Println("Size of Connection Pool: ", len(pool.Clients), client.ID, " dropped off.")
			//}
			break
		}
	}
}

func (pool *Pool) StartBroadcast() {
	const ErrorText = "Broadcast update error - %s\n"
	for {
		select {
		case message := <-pool.Broadcast:
			//if debugOutput {
			//	log.Printf("message received for service - %d (device = [%s]", message.service, message.device)
			//}
			for client := range pool.Clients {
				if client.Service == message.service {
					// Client has requested this service
					if !(message.service == wsElectrolyser && client.Device != message.device) {
						// This is either the right electrolyser or the service is not wsElectrolyser
						// We will give it 3 seconds to complete
						// If we hit an error, log it and drop the client. The client will have to reconnect.
						if err := client.Conn.SetWriteDeadline(time.Now().Add((time.Second * 3))); err != nil {
							log.Printf(ErrorText, err)
							delete(pool.Clients, client)
						} else {
							if err := client.Conn.WriteMessage(websocket.TextMessage, message.data); err != nil {
								log.Printf(ErrorText, err)
								delete(pool.Clients, client)
							}
						}
					}
				}
			}
			break
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return conn, nil
}
