package main

import (
	"log"
	"net/http"
)

func main() {
	studentHandler := studentHttpHandler()
	sessionsHandler := sessionHttpHandler()
	classesHandler := classHttpHandler()
	teacherHandler := teacherHttpHandler()
	http.Handle("/students", studentHandler)
	http.Handle("/sessions", sessionsHandler)
	http.Handle("/classes", classesHandler)
	http.Handle("/teachers", teacherHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
