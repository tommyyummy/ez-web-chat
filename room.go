package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"golang.org/x/exp/slices"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
)

type Message struct {
	Username  string `json:"username"`
	Message   string `json:"message"`
	Timestamp int64  `json:"ts"`
}

type MyChatDB struct {
	RawDB        *bolt.DB
	Rooms        map[string]map[*websocket.Conn]bool
	RoomChannels map[string](chan Message)
}

var MyDB *MyChatDB

func DBInit() {
	db, err := bolt.Open("chat.db", 0600, &bolt.Options{})
	if err != nil {
		log.Fatal(err)
	}
	MyDB = &MyChatDB{
		RawDB:        db,
		Rooms:        make(map[string]map[*websocket.Conn]bool, 0),
		RoomChannels: make(map[string](chan Message), 0),
	}
	MyDB.initRooms()
}

func (m *MyChatDB) roomStart(room string) {
	roomChannel, ok := m.RoomChannels[room]
	if !ok {
		return
	}
	fmt.Println("room started", room)
	for {
		msg := <-roomChannel

		clients, ok := m.Rooms[room]
		if ok {
			for client := range clients {
				err := client.WriteJSON(msg)
				if err != nil {
					fmt.Println(err)
					client.Close()
					delete(clients, client)
				}
			}
		}
	}
}

func (m *MyChatDB) initRooms() {
	_ = m.RawDB.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			m.Rooms[string(name)] = make(map[*websocket.Conn]bool, 0)
			m.RoomChannels[string(name)] = make(chan Message)
			go m.roomStart(string(name))
			return nil
		})
		return nil
	})
}

func (m *MyChatDB) close() {
	m.RawDB.Close()
}

func (m *MyChatDB) CreateRoom(room string) error {
	err := m.RawDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(room))
		return err
	})
	if err != nil {
		return err
	}
	m.Rooms[room] = make(map[*websocket.Conn]bool, 0)
	m.RoomChannels[room] = make(chan Message)
	go m.roomStart(room)
	return nil
}

const chatLimit int64 = 20

func getSingleMessage(res []Message, bucket *bolt.Bucket, index int64) []Message {
	s := bucket.Get([]byte(fmt.Sprintf("chat-content:%d", index)))
	if string(s) != "" {
		data := Message{}
		e := json.Unmarshal(s, &data)
		if e == nil {
			res = append(res, data)
		}
	}
	return res
}

func (m *MyChatDB) GetMessages(room string) []Message {
	_, ok := m.Rooms[room]
	if !ok {
		return make([]Message, 0)
	}

	res := make([]Message, 0)

	err := m.RawDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(room))
		if b == nil {
			return fmt.Errorf("get bucket: FAILED")
		}
		index, _ := strconv.ParseInt(string(b.Get([]byte("chat-index"))), 10, 64)
		for i := index - 1; i >= 0; i-- {
			res = getSingleMessage(res, b, i)
		}
		for i := chatLimit - 1; i >= index; i-- {
			res = getSingleMessage(res, b, i)
		}
		return nil
	})
	if err != nil {
		return make([]Message, 0)
	}
	slices.Reverse(res)
	return res
}

func (m *MyChatDB) AddMessage(room string, msg *Message) error {
	_, ok := m.Rooms[room]
	if !ok {
		return fmt.Errorf("invalid room")
	}

	content, _ := json.Marshal(msg)

	err := m.RawDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(room))
		if b == nil {
			return fmt.Errorf("get bucket: FAILED")
		}
		index, _ := strconv.ParseInt(string(b.Get([]byte("chat-index"))), 10, 64)

		if b.Put([]byte(fmt.Sprintf("chat-content:%d", index)), content) != nil {
			return fmt.Errorf("update chat content FAILED")
		}

		newIndex := (index + 1) % chatLimit
		if b.Put([]byte("chat-index"), []byte(fmt.Sprintf("%d", newIndex))) != nil {
			return fmt.Errorf("update chat index FAILED")
		}
		return nil
	})

	return err
}
