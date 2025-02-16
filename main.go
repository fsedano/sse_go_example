package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/events/{id}", eventsHandler)

	go func() {
		http.ListenAndServe(":8080", mux)
	}()

	// simulate send data
	for {
		log.Println("Round to simulate")
		for idx := 0; idx <= 10; idx++ {
			clientId := fmt.Sprintf("%d", idx)
			if clientList, found := clientMap[clientId]; found {
				cd := clientData{Name: "perico", Age: idx}
				log.Printf("Found data for client %v: %v", cd, clientList)
				for _, client := range clientList {
					client <- cd
				}

			}
		}
		time.Sleep(2 * time.Second)
	}
}

type clientData struct {
	Name string
	Age  int
}

var clients = make(map[chan clientData]bool)
var clientMap = make(map[string][]chan clientData)

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	log.Printf("We are %s", id)
	// Set CORS headers to allow all origins. You may want to restrict this to specific origins in a production environment.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	done := false
	var c = make(chan bool)
	var d = make(chan clientData)

	clients[d] = true
	//clientMap[id] = d
	// Add ourselves to client list for this id
	// Todo lock
	clientList := clientMap[id]
	clientList = append(clientList, d)
	clientMap[id] = clientList
	go func() {
		sendData(w, d, c)
	}()

	for !done {
		select {
		case <-r.Context().Done():
			// Client gave up
			log.Printf("Client disconnected")
			done = true
			delete(clients, d)
			// Delete ourselves from clientList. TODO lock
			//delete(clientMap, id)
			clientList := clientMap[id]
			var newClientList []chan clientData
			for _, el := range clientList {
				if el != d {
					newClientList = append(newClientList, el)
				}
			}
			clientMap[id] = newClientList
			// Unlock
			c <- true
		default:
		}
	}
	log.Printf("Client closed")
}

func sendData(w http.ResponseWriter, d chan clientData, c chan bool) {
	for {
		select {
		case <-c:
			log.Printf("We die")
			return
		case data := <-d:
			dataJson, _ := json.MarshalIndent(data, "", "  ")
			log.Printf("data=%s", dataJson)
			fmt.Fprintf(w, "data: %s\n\n", dataJson)
			w.(http.Flusher).Flush()
		}
	}
}
