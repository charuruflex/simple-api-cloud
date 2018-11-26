package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

var users map[string]string

func updateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var body map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err := decoder.Decode(&body)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusBadRequest)
		return
	case body["dateOfBirth"] == nil:
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("missing argument dateOfBirth"))
		return
	}

	birthday := body["dateOfBirth"].(string)
	// birthday, _ := time.Parse(time.RFC3339, body["dateOfBirth"].(string))
	users[vars["username"]] = birthday
	fmt.Printf("creating/updating user %s %v\n", vars["username"], users[vars["username"]])
	w.WriteHeader(http.StatusNoContent)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	u, b := vars["username"], users[vars["username"]]
	fmt.Printf("getting user %s, birthday: %s\n", u, b)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{fmt.Sprintf("Hello, %s! Your birthday is %s", u, b)})
}

func main() {
	fmt.Println("starting app")

	users = make(map[string]string)

	r := mux.NewRouter()

	r.HandleFunc("/hello/{username}", getUser).Methods("GET")
	r.HandleFunc("/hello/{username}", updateUser).Methods("PUT")

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	srv := &http.Server{
		Addr: "0.0.0.0:8000",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
