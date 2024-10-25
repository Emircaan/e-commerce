package main

import (
	"log"

	"github.com/Emircaan/e-commerce/db"
)

func main() {
	db, err := db.NewDatabase()
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to database")

}
