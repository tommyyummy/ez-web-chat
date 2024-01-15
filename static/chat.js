console.log("JS Loaded")

const searchParams = new URLSearchParams(window.location.search);
room = searchParams.get('room')

let socket = new WebSocket("ws://127.0.0.1:8080/ws?room=" + room);

// send message from the form
document.forms.publish.onsubmit = function() {
    let outgoingMessage = this.message.value;
    let user = this.user.value

    if (!outgoingMessage || !user) {
        return false
    }

    let data = {
        "room": room,
        "username": user,
        "message": outgoingMessage,
    }

    console.log(socket.readyState)
    console.log(data)

    socket.send(JSON.stringify(data));
    return false;
  };

socket.onmessage = event => {
    console.log(event)

    let data = JSON.parse(event.data)
    let content = data.username + " : " + data.message + " : ts " + data.ts
    console.log("content", content)

    let messageElement = document.createElement('div')
    messageElement.textContent = content
    document.getElementById("messages").prepend(messageElement)
}

socket.onclose = () => {
    console.log("closed")
    let ws = document.getElementById("wsStatus")
    ws.textContent = "CONNECTION STATUS: CLOSED"
    document.getElementById("sendButton").disabled = true 
}

socket.onopen = () => {
    console.log("opened")
    let ws = document.getElementById("wsStatus")
    ws.textContent = "CONNECTION STATUS: OPEN"
    document.getElementById("sendButton").disabled = false
}