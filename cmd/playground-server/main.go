package main

import (
	"context"
	"log"
	"net/http"
	"sync"

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
	game.LoadROMFromFile("test-roms/6-keypad.ch8")
	game.Start(context.Background())
}

type KeyEventType int

const (
	KEY_UP KeyEventType = iota
	KEY_DOWN
)

type KeyEvent struct {
	key string
	tp  KeyEventType
}

type Keys uint8

const (
	KEY_0 Keys = iota
	KEY_1
	KEY_2
	KEY_3
	KEY_4
	KEY_5
	KEY_6
	KEY_7
	KEY_8
	KEY_9
	KEY_A
	KEY_B
	KEY_C
	KEY_D
	KEY_E
	KEY_F
)

type PlaygroundProvider struct {
	mu    sync.Mutex
	c     *websocket.Conn
	inbox <-chan KeyEvent
	keys  [16]bool
}

func (p *PlaygroundProvider) keyProcessor() {
	for {
		event := <-p.inbox
		var code uint8
		switch event.key {
		case "0":
			code = uint8(KEY_0)
		case "1":
			code = uint8(KEY_1)
		case "2":
			code = uint8(KEY_2)
		case "3":
			code = uint8(KEY_3)
		case "4":
			code = uint8(KEY_4)
		case "5":
			code = uint8(KEY_5)
		case "6":
			code = uint8(KEY_6)
		case "7":
			code = uint8(KEY_7)
		case "8":
			code = uint8(KEY_8)
		case "9":
			code = uint8(KEY_9)
		case "A":
			code = uint8(KEY_A)
		case "B":
			code = uint8(KEY_B)
		case "C":
			code = uint8(KEY_C)
		case "D":
			code = uint8(KEY_D)
		case "E":
			code = uint8(KEY_E)
		case "F":
			code = uint8(KEY_F)
		}
		p.mu.Lock()
		if event.tp == KEY_DOWN {
			p.keys[code] = true
		} else {
			p.keys[code] = false
		}
		p.mu.Unlock()
	}
}

func NewPlaygroundProvider(c *websocket.Conn) *PlaygroundProvider {
	ch := make(chan KeyEvent, 1024)
	go jsonReader(c, ch)
	obj := PlaygroundProvider{c: c, inbox: ch, mu: sync.Mutex{}}
	go obj.keyProcessor()
	return &obj
}

func jsonReader(c *websocket.Conn, ch chan<- KeyEvent) {
	for {
		data := map[string]string{}
		err := c.ReadJSON(&data)
		if err != nil {
			log.Println("read json:", err)
			close(ch)
			c.Close()
			return
		} else {
			log.Printf("data %+v\n", data)
			if tp, ok1 := data["type"]; ok1 {
				if key, ok2 := data["data"]; ok2 && (tp == "key.down" || tp == "key.up") {
					k := KeyEvent{}
					if tp == "key.down" {
						k.tp = KEY_DOWN
					} else {
						k.tp = KEY_UP
					}
					k.key = key
					log.Printf("key %+v\n", key)
					ch <- k
				}
			}
		}
	}
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
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, state := range p.keys {
		if state {
			return uint8(key), true
		}
	}

	return 0, false
}

// IsKeyPressed implements [vm.KeyboardProvider].
func (p *PlaygroundProvider) IsKeyPressed(key uint8) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.keys[key]
}

var _ vm.KeyboardProvider = (*PlaygroundProvider)(nil)
var _ vm.DisplayProvider = (*PlaygroundProvider)(nil)
var _ vm.SoundProvider = (*PlaygroundProvider)(nil)
