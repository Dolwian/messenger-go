package main

import (
	"log"
	"net/http"

	"github.com/dolwian/messenger-go/internal/ws"
)

func file(path string) http.HandlerFunc {
	path = "web/" + path
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path)
	}
}

func main() {
	log.Println("Listening on :8080")
	hub := ws.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")

		conn, err := ws.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := ws.NewClient(hub, conn, userID)

		hub.Register(client)

		// Запускаем горутины для чтения и записи
		go client.WritePump()
		go client.ReadPump()
	})
	http.HandleFunc("/", file("static/index.html"))
	http.HandleFunc("/style.css", file("static/style.css"))
	http.ListenAndServe(":8080", nil)
}
