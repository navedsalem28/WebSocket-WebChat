package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var upgrade = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// You can implement custom logic here to check the origin.
		// Return true if the origin is allowed, and false otherwise.
		return true // For allowing all origins, but customize this as needed.
	},
}

type ClientList struct {
	webSocketConnection *websocket.Conn
	name                string
	port                string
}

var Clients []*ClientList

func main() {

	router := mux.NewRouter()

	// Set up CORS handling
	headers := handlers.AllowedHeaders([]string{"*"})
	methods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
	origins := handlers.AllowedOrigins([]string{"*"}) // Allow requests from any origin

	// Add CORS middleware
	router.Use(handlers.CORS(headers, methods, origins))

	// Define your routes here
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	router.HandleFunc("/login", login)
	go router.HandleFunc("/echo", ClientSocket)

	// Start your server
	err := http.ListenAndServe(":8081", router)
	if err != nil {
		log.Println("Start your server Error: ", err.Error())
		return
	}
}

// login API use according to your requirement
func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("login")
	RequestJSON := make(map[string]interface{})
	Response := make(map[string]interface{})
	BodyText, err := ioutil.ReadAll(r.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(r.Body)
	if err != nil {
		Response["message"] = "Invalid Request Body"
		Response["status"] = 500
		fmt.Println("processAPI: can't ready request body text, Error: " + err.Error())
		err := json.NewEncoder(w).Encode(Response)
		if err != nil {
			return
		}
		return
	}
	fmt.Println("Request Body Text:\t" + string(BodyText))
	err = json.Unmarshal(BodyText, &RequestJSON)

	fmt.Println(RequestJSON)

	Response["responseCode"] = 200
	Response["responseMessage"] = "accepted"
	responseBytes, _ := json.Marshal(Response)
	_, _ = fmt.Fprintf(w, "%v", string(responseBytes))
}

// ClientSocket Accept new client request
func ClientSocket(w http.ResponseWriter, r *http.Request) {

	Conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func(Conn *websocket.Conn) {
		err := Conn.Close()
		if err != nil {

		}
	}(Conn)
	tick := time.NewTicker(5)

	for {
		messageType, requestByte, err := Conn.ReadMessage()

		if err != nil {
			log.Println("ERROR read", err)
			SocketDrop(messageType, Conn)
			break
		}

		requestJson := make(map[string]interface{})
		err = json.Unmarshal(requestByte, &requestJson)
		if err != nil {
			log.Println("Unknown Method: ", err.Error())
			return
		}
		fmt.Println("Client_msg", requestJson)
		switch requestJson["requestType"] {
		case "register":
			Register(requestJson, messageType, Conn)
			break
		case "msg":
			message(requestJson, messageType, Conn)
		case "status":
			status(messageType, Conn)
		case tick.C:
			ping(messageType, Conn)
		default:
			//	TODO: Handle unknown request and update client
			log.Println("Unknown type: ", requestJson["requestType"])
			break
		}
	}

}

// SocketDrop : Delete Client and Session
func SocketDrop(messageType int, Conn *websocket.Conn) {
	x := strings.Split(Conn.RemoteAddr().String(), ":")
	port := x[3]
	for index, client := range Clients {
		if client.port == port {
			Clients = append(Clients[:index], Clients[index+1:]...)
			fmt.Println("SOCKET Is DROPPED By: ", Conn.RemoteAddr())
		}
	}
	Response := make(map[string]interface{})
	var List []string
	for _, v := range Clients {
		List = append(List, v.name)
	}
	Response["List"] = List
	responseBytes, _ := json.Marshal(Response)
	WriteOnSocket(responseBytes, messageType, Conn)
}

// Register Client on Web Socket
func Register(RequestJSON map[string]interface{}, messageType int, Conn *websocket.Conn) {
	flag := false
	name := RequestJSON["name"]
	x := strings.Split(Conn.RemoteAddr().String(), ":")
	port := x[3]
	Response := make(map[string]interface{})
	for _, v := range Clients {

		if v.port == port {
			flag = true
			port = v.port
			name = v.name
		}
	}

	if flag == false {
		client := &ClientList{
			webSocketConnection: Conn,
			name:                InterfaceToString(name),
			port:                InterfaceToString(port),
		}
		Clients = append(Clients, client)
		Response["responseCode"] = 200
		Response["responseMessage"] = "accepted"
	} else {
		Response["responseMessage"] = "already_registered"
	}
	var List []string
	for _, v := range Clients {
		List = append(List, v.name)
	}
	Response["List"] = List
	Response["responseMessage"] = "pong"
	responseBytes, _ := json.Marshal(Response)
	WriteOnSocket(responseBytes, messageType, Conn)
}

func WriteOnSocket(responseBytes []byte, messageType int, Conn *websocket.Conn) {
	//time.Sleep(2)

	fmt.Println("")
	fmt.Println("")
	fmt.Println("====================")
	log.Printf("write: %s", responseBytes)
	fmt.Println("====================")
	fmt.Println("")
	fmt.Println("")

	err := Conn.WriteMessage(messageType, responseBytes)
	if err != nil {
		log.Println("write:", err)
	}
}

func message(RequestJSON map[string]interface{}, messageType int, Conn *websocket.Conn) {
	var name string
	x := strings.Split(Conn.RemoteAddr().String(), ":")
	port := x[3]
	var List []string
	for _, v := range Clients {
		List = append(List, v.name)
		if v.port == port {
			name = v.name
		}
	}
	Response := make(map[string]interface{})
	Response["List"] = List
	Response["msg"] = RequestJSON["msg"]
	Response["name"] = name
	Response["responseMessage"] = "message"
	responseBytes, _ := json.Marshal(Response)
	for _, v := range Clients {
		WriteOnSocket(responseBytes, messageType, v.webSocketConnection)
	}

}

// InterfaceToString (interface{}) get value of an interface return as string {
func InterfaceToString(ThisInterface interface{}) string {
	if ThisInterface == nil {
		fmt.Println("InterfaceToString :: NIL value passed for conversion")
		return ""
	}
	switch ThisInterface.(type) {
	case int:
		return strconv.Itoa(ThisInterface.(int))
	case int64:
		return strconv.FormatInt(ThisInterface.(int64), 10)
	case float64:
		TmpStr := strconv.FormatFloat(ThisInterface.(float64), 'f', 10, 64)
		if TmpStr[(len(TmpStr)-10):] == "0000000000" {
			return TmpStr[:(len(TmpStr) - 11)]
		}
		return TmpStr
	default:
		return ThisInterface.(string)
	}
}

//status get the Active users list
func status(messageType int, Conn *websocket.Conn) {
	var List []string
	for _, v := range Clients {
		List = append(List, v.name)
	}
	Response := make(map[string]interface{})
	Response["List"] = List
	Response["responseMessage"] = "pong"
	responseBytes, _ := json.Marshal(Response)
	WriteOnSocket(responseBytes, messageType, Conn)
}

//ping maintain connection state
func ping(messageType int, Conn *websocket.Conn) {
	var List []string
	for _, v := range Clients {
		List = append(List, v.name)
	}
	Response := make(map[string]interface{})
	Response["List"] = List
	Response["responseMessage"] = "pong"
	responseBytes, _ := json.Marshal(Response)
	WriteOnSocket(responseBytes, messageType, Conn)
}
