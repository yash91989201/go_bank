package main

import (
	"log"
)

func main() {
	store, err := newPostgresStore()

	if err != nil {
		log.Fatalf("creating postgres store %v", err)
	}

	store.CreateAccountTable()

	server := NewAPIServer(":8000", store)
	server.Run()
}
