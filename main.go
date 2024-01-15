package main

import (
	"fmt"
	"net/http"
)

func main() {

	DBInit()
	defer MyDB.close()

	http.HandleFunc("/", defaultResponse)
	http.HandleFunc("/new-room", createNewRoom)
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/chat", handleChatPage)

	fmt.Println("Server starting on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("Error starting server: " + err.Error())
	}
}
