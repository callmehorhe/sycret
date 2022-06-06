package main

import "log"

func main() {
	srv := Serv{}
	if err := srv.Run(":8000"); err != nil {
		log.Fatal(err)
	}
}
