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
	"github.com/mediocregopher/radix"
)

var pool, err = radix.NewPool("tcp", "127.0.0.1:6379", 10)

const dateFormat = "2006-01-02"

type msg struct {
	Message string `json:"message"`
}

type bDay struct {
	Birthday *time.Time `json:"dateOfBirth"`
}

func (b *bDay) UnmarshalJSON(d []byte) error {
	var bDayTmp struct {
		Birthday string `json:"dateOfBirth"`
	}

	err := json.Unmarshal(d, &bDayTmp)
	if err != nil {
		fmt.Println(err)
		return err
	}

	t, err := time.Parse(dateFormat, bDayTmp.Birthday)
	if err != nil {
		fmt.Println(err)
		return err
	}

	b.Birthday = &t

	return nil
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var bd bDay

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err := decoder.Decode(&bd)
	switch {
	case err != nil:
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	case bd.Birthday == nil:
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("missing argument dateOfBirth"))
		return
	}

	pool.Do(radix.Cmd(nil, "SET", vars["username"], (*bd.Birthday).Format(dateFormat)))
	fmt.Println("creating/updating user", vars["username"], (*bd.Birthday).Format(dateFormat))
	w.WriteHeader(http.StatusNoContent)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	u := vars["username"]
	var bs string
	pool.Do(radix.Cmd(&bs, "GET", u))

	b, _ := time.Parse(dateFormat, bs)
	fmt.Printf("getting user %s, birthday: %s\n", u, b.Format(dateFormat))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg{fmt.Sprintf("Hello, %s! Your birthday is %s", u, b.Format(dateFormat))})

	// TODO: add case when user doesn't exist
}

func main() {
	fmt.Println("starting app")

	// pool, err = radix.NewPool("tcp", "127.0.0.1:6379", 10)
	// if err != nil {
	// 	// handle error
	// }

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
