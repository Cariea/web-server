package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID     string
	Name   string
	Socket *websocket.Conn
}

type Game struct {
	Players []Player
}

type playersWithoutSocket struct {
	ID   string
	Name string
}

type Message struct {
	Event    string
	Data     string
	Sender   string
	Reciever string
}

var players []Player
var games []Game

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				if closeErr.Code == websocket.CloseNormalClosure {
					fmt.Println("Cliente cerró la conexión normalmente")
					for i, player := range players {
						if player.Socket == conn {
							players = append(players[:i], players[i+1:]...)
							break
						}
					}
				} else {
					fmt.Println("Cliente cerró la conexión inesperadamente:", closeErr.Error())
					for i, player := range players {
						if player.Socket == conn {
							players = append(players[:i], players[i+1:]...)
							break
						}
					}
				}
				break
			}
			log.Println("Error al leer mensaje del cliente:", err)
			for i, player := range players {
				if player.Socket == conn {
					players = append(players[:i], players[i+1:]...)
					break
				}
			}
			break
		}

		for game := range games {
			if games[game].Players[0].Socket == conn {
				games[game].Players[1].Socket.WriteMessage(websocket.TextMessage, message)
				break
			} else if games[game].Players[1].Socket == conn {
				games[game].Players[0].Socket.WriteMessage(websocket.TextMessage, message)
				break
			}
		}

	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)

	var player = Player{ID: r.Header.Get("id"), Name: r.Header.Get("name"), Socket: conn}
	players = append(players, player)

	if err != nil {
		log.Println("Error al actualizar la conexión WebSocket:", err)
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte("pinghola"))

	fmt.Println("Cliente conectado:", conn.RemoteAddr())
	go handleWebSocket(conn)
}

// devolver jugadores en el response
func getPlayers(w http.ResponseWriter, r *http.Request) {
	var playersw []playersWithoutSocket
	for _, player := range players {
		playersw = append(playersw, playersWithoutSocket{ID: player.ID, Name: player.Name})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playersw)
}

func handleGame(w http.ResponseWriter, r *http.Request) {

	var playerOneId = r.Header.Get("playerOneId")
	var playerTwoId = r.Header.Get("playerTwoId")
	var playerOneName = r.Header.Get("playerOneName")

	var playerOneSocket *websocket.Conn
	var playerTwoSocket *websocket.Conn

	for _, player := range players {

		if player.ID == playerOneId {
			playerOneSocket = player.Socket
		}
		if player.ID == playerTwoId {
			playerTwoSocket = player.Socket
		}

	}
	var playerOneNamelength = strconv.Itoa(len(playerOneName))

	if playerOneSocket != nil && playerTwoSocket != nil {
		games = append(games, Game{Players: []Player{{ID: playerOneId, Socket: playerOneSocket}, {ID: playerTwoId, Socket: playerTwoSocket}}})

	}

	playerTwoSocket.WriteMessage(websocket.TextMessage, []byte("init"+playerOneNamelength+playerOneName))

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode("Conexión establecida")
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/players", getPlayers)
	http.HandleFunc("/game", handleGame)

	err := http.ListenAndServe(":52301", nil)
	if err != nil {
		log.Fatal("Error al iniciar el servidor:", err)
	}

}
