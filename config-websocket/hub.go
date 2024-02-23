package configwebsocket

import "fmt"

type Hub struct {
	rooms       map[string]map[*connection]bool
	broadcast   chan message
	register    chan subscription
	unregister  chan subscription
	activeConns map[string]int
}

type message struct {
	Room string `json:"room"`
	Data []byte `json:"data"`
}

var H = &Hub{
	broadcast:   make(chan message),
	register:    make(chan subscription),
	unregister:  make(chan subscription),
	rooms:       make(map[string]map[*connection]bool),
	activeConns: make(map[string]int),
}

func (h *Hub) Run() {
	fmt.Println("Hub running")
	for {
		select {
		case s := <-h.register:
			connections := h.rooms[s.room]
			if connections == nil {
				connections = make(map[*connection]bool)
				h.rooms[s.room] = connections
			}
			connections[s.conn] = true
			h.activeConns[s.room]++
			fmt.Println("connection", h.activeConns)

		case s := <-h.unregister:
			fmt.Println(s.room)
			h.activeConns[s.room]--
			if h.activeConns[s.room] == 0 {
				delete(h.rooms, s.room)
				delete(h.activeConns, s.room)
			}
			fmt.Println("no", h.activeConns)

		case m := <-h.broadcast:
			connections := h.rooms[m.Room]
			for c := range connections {
				select {
				case c.send <- m.Data:
				default:
					delete(connections, c)
					close(c.send)
					h.activeConns[m.Room]--
				}
			}
		}
	}
}