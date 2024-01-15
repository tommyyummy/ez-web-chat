package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func defaultResponse(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// fmt.Println(path)
	if strings.HasPrefix(path, "/static") {
		path = "." + path
		http.ServeFile(w, r, path)
		return
	}
	homePage(w, r)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/html")

	fmt.Fprintf(w, `
	<html>
	<title>[EZ Chat]</title>
	<div style='text-align:left'>
	<h1>chat rooms</h1>
	<style>
	.cell { display: inline-block; width: 100em; }
	</style>
	</div>
	<body>
	<form>
	<input type="text" name="room">
	<input type="submit" value="Create New Room" formaction=/new-room>
	</form>
	`)

	for room := range MyDB.Rooms {
		fmt.Fprintf(w, `
		<div class=cell>
		<div><a class=key href='/chat?room=%s'>%s</div>
		</div>
		`, room, room)
	}

	fmt.Fprintf(w, `
	</body>
	`)
}

func createNewRoom(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	MyDB.CreateRoom(room)
	homePage(w, r)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	room := r.URL.Query().Get("room")

	clients, ok := MyDB.Rooms[room]

	if !ok {
		return
	}

	broadcast, ok := MyDB.RoomChannels[room]
	if !ok {
		return
	}

	clients[conn] = true

	for _, m := range MyDB.GetMessages(room) {
		err := conn.WriteJSON(m)
		if err != nil {
			fmt.Println(err)
			delete(clients, conn)
			return
		}
	}

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println(err)
			delete(clients, conn)
			return
		}
		msg.Timestamp = time.Now().Unix()

		err = MyDB.AddMessage(room, &msg)
		if err == nil {
			broadcast <- msg
		} else {
			fmt.Println(err.Error())
		}
	}
}

var tmpl *template.Template

type ChatRoomData struct {
	RoomName string
}

func handleChatPage(w http.ResponseWriter, r *http.Request) {
	tmpl = template.Must(template.ParseFiles("./static/chat.html"))
	room := r.URL.Query().Get("room")

	if _, ok := MyDB.Rooms[room]; !ok {
		defaultResponse(w, r)
		return
	}

	data := &ChatRoomData{RoomName: room}
	if err := tmpl.Execute(w, data); err != nil {
		log.Print(err)
	}
}
