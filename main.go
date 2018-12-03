package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/mediocregopher/radix"
	yaml "gopkg.in/yaml.v2"
)

var pool *radix.Pool
var conf config
var version = "new"

type config struct {
	Database struct {
		Redis string
	}
	Server struct {
		Addr         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}
}

const dateFormat = "2006-01-02"

type msg struct {
	Message string `json:"message"`
}

type bDay struct {
	DateOfBirth *time.Time `json:"dateOfBirth"`
}

func (b *bDay) UnmarshalJSON(d []byte) error {
	var bDayTmp struct {
		DateOfBirth string `json:"dateOfBirth"`
	}

	err := json.Unmarshal(d, &bDayTmp)
	if err != nil {
		fmt.Println(err)
		return err
	}

	t, err := time.Parse(dateFormat, bDayTmp.DateOfBirth)
	if err != nil {
		fmt.Println(err)
		return err
	}

	b.DateOfBirth = &t

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
		json.NewEncoder(w).Encode(msg{fmt.Sprintf("bad date format for dateOfBirth: %v", err)})
		return
	case bd.DateOfBirth == nil:
		w.WriteHeader(http.StatusNotAcceptable)
		json.NewEncoder(w).Encode(msg{"missing argument dateOfBirth"})
		return
	}

	pool.Do(radix.Cmd(nil, "SET", vars["username"], (*bd.DateOfBirth).Format(dateFormat)))
	fmt.Println("creating/updating user", vars["username"], (*bd.DateOfBirth).Format(dateFormat))
	w.WriteHeader(http.StatusNoContent)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	u := vars["username"]
	var DOFS string
	pool.Do(radix.Cmd(&DOFS, "GET", u))

	if DOFS == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(msg{fmt.Sprintf("Hello! Unfortunately I don't know %s yet. Please add his/her date of birth.", u)})
		return
	}

	dateOfBirth, _ := time.Parse(dateFormat, DOFS)
	fmt.Printf("getting user %s, date of birth: %s\n", u, dateOfBirth.Format(dateFormat))

	now := time.Now()
	birthday := time.Date(now.Year(), dateOfBirth.Month(), dateOfBirth.Day(), 0, 0, 0, 0, now.Location())
	if now.After(birthday) {
		birthday = time.Date(now.Year()+1, dateOfBirth.Month(), dateOfBirth.Day(), 0, 0, 0, 0, now.Location())
	}
	daysBeforeBDay := int(birthday.Sub(now).Round(time.Hour).Hours()) / 24

	switch {
	case daysBeforeBDay == 0:
		json.NewEncoder(w).Encode(msg{fmt.Sprintf("Hello, %s! Happy birthday!", u)})
	case daysBeforeBDay == 1:
		json.NewEncoder(w).Encode(msg{fmt.Sprintf("Hello, %s! Your birthday is in %d day", u, daysBeforeBDay)})
	default:
		json.NewEncoder(w).Encode(msg{fmt.Sprintf("Hello, %s! Your birthday is in %d days", u, daysBeforeBDay)})
	}

}

func info(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	hostname, _ := os.Hostname()
	json.NewEncoder(w).Encode(map[string]string{"app": "simple-api-cloud", "version": version, "hostname": hostname})
}

func loadConfig(file string) (cfg config, err error) {
	cfg.Server.Addr = ":8080"
	cfg.Server.WriteTimeout = time.Second * 15
	cfg.Server.ReadTimeout = time.Second * 15
	cfg.Server.IdleTimeout = time.Second * 60
	cfg.Database.Redis = fmt.Sprintf("%s:%s", os.Getenv("REDIS_MASTER_SERVICE_HOST"), os.Getenv("REDIS_MASTER_SERVICE_PORT"))

	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Println(err)
		return
	}

	return
}

func main() {
	fmt.Println("starting app")
	filename := flag.String("config", "config.yml", "Configuration file")
	flag.Parse()

	conf, err := loadConfig(*filename)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	pool, err = radix.NewPool("tcp", conf.Database.Redis, 10)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	r := mux.NewRouter()
	r.HandleFunc("/", info).Methods("GET")
	r.HandleFunc("/hello/{username}", getUser).Methods("GET")
	r.HandleFunc("/hello/{username}", updateUser).Methods("PUT")

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	srv := &http.Server{
		Addr: fmt.Sprintf("%s", conf.Server.Addr),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: conf.Server.WriteTimeout,
		ReadTimeout:  conf.Server.ReadTimeout,
		IdleTimeout:  conf.Server.IdleTimeout,
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
