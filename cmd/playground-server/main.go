package main

import (
	"context"
	"log"
	"net/http"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/gorilla/websocket"
)

func main() {
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/heartbit", heartbit)
	err := http.ListenAndServe("localhost:9000", nil)
	log.Fatal(err)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func heartbit(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	pg := NewPlaygroundProvider(c)
	game := vm.NewChip8VM(pg, pg, pg)
	game.LoadROMFromFile("test-games/heart_monitor.ch8")
	game.Start(context.Background())
}

type PlaygroundProvider struct {
	c *websocket.Conn
}

func NewPlaygroundProvider(c *websocket.Conn) *PlaygroundProvider {
	return &PlaygroundProvider{c: c}
}

// PlaySound implements [vm.SoundProvider].
func (p *PlaygroundProvider) PlaySound() {
	message := map[string]string{}
	message["type"] = "sound.play"
	p.c.WriteJSON(message)
}

// StopSound implements [vm.SoundProvider].
func (p *PlaygroundProvider) StopSound() {
	message := map[string]string{}
	message["type"] = "sound.stop"
	p.c.WriteJSON(message)
}

// Clear implements [vm.DisplayProvider].
func (p *PlaygroundProvider) Clear() {
	message := map[string]string{}
	message["type"] = "display.clear"
	p.c.WriteJSON(message)
}

// Draw implements [vm.DisplayProvider].
func (p *PlaygroundProvider) Draw(data [64][32]bool) {
	message := map[string]any{}
	message["type"] = "display.draw"
	message["data"] = data
	p.c.WriteJSON(message)
}

// Close implements [vm.KeyboardProvider].
func (p *PlaygroundProvider) Close() {
}

// GetPressedKey implements [vm.KeyboardProvider].
func (p *PlaygroundProvider) GetPressedKey() (key uint8, pressed bool) {
	return 0, false
}

// IsKeyPressed implements [vm.KeyboardProvider].
func (p *PlaygroundProvider) IsKeyPressed(key uint8) bool {
	return false
}

var _ vm.KeyboardProvider = (*PlaygroundProvider)(nil)
var _ vm.DisplayProvider = (*PlaygroundProvider)(nil)
var _ vm.SoundProvider = (*PlaygroundProvider)(nil)
