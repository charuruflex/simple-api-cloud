package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/mediocregopher/radix"
)

func TestUnmarshalBDay(t *testing.T) {
	d := "2018-11-26"
	input := []byte(fmt.Sprintf(`{"dateOfBirth":"%s"}`, d))
	var b bDay

	err := json.Unmarshal(input, &b)
	if err != nil {
		t.Errorf("error on unmarshalling %v: %v", input, err)
	}

	if bd := b.DateOfBirth.Format(dateFormat); bd != d {
		t.Errorf("got unexpected date: got %s, want %s", bd, d)
	}
}

func initLocalRedis(rType string) (err error) {
	localRedis := "127.0.0.1:6379"

	switch rType {
	case "master":
		redisMaster, err = radix.NewPool("tcp", localRedis, 10)
	case "slave":
		redisSlave, err = radix.NewPool("tcp", localRedis, 10)

	}

	if err != nil {
		err = fmt.Errorf("fail to connect to local redis %s: %v", localRedis, err)
	}

	return
}

func TestUpdateUser(t *testing.T) {
	if err := initLocalRedis("master"); err != nil {
		t.Error(err)
	}
	defer redisMaster.Close()

	d := "2018-11-26"
	input := fmt.Sprintf(`{"dateOfBirth":"%s"}`, d)
	req, err := http.NewRequest("PUT", "/hello/toto", strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/hello/{username}", updateUser)
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("unexpected response code: got %v, want %v", status, http.StatusNoContent)
	}

}

func TestGetExistingUser(t *testing.T) {
	if err := initLocalRedis("slave"); err != nil {
		t.Error(err)
	}
	defer redisSlave.Close()

	d := time.Now().Format("2006-01-02")
	redisSlave.Do(radix.Cmd(nil, "SET", "toto", d))

	req, err := http.NewRequest("GET", "/hello/toto", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/hello/{username}", getUser)
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user not found, unexpected response code: got %v, want %v", status, http.StatusNoContent)
	}

	var rrMsg msg
	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Errorf("error on getting body %v", err)
	}
	err = json.Unmarshal(body, &rrMsg)

	if rrMsg.Message != "Hello, toto! Your birthday is in 364 days" {
		t.Errorf("unexpected date: got %s, want %s", rrMsg.Message, d)
	}
}

func TestGetNonExistingUser(t *testing.T) {
	if err := initLocalRedis("slave"); err != nil {
		t.Error(err)
	}
	defer redisSlave.Close()

	redisSlave.Do(radix.Cmd(nil, "DEL", "toto"))

	req, err := http.NewRequest("GET", "/hello/toto", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/hello/{username}", getUser)
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("user not found, unexpected response code: got %v, want %v", status, http.StatusNoContent)
	}
}

func TestInfo(t *testing.T) {
	req, err := http.NewRequest("GET", "/info", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/info", info)
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("unexpected response code: got %v, want %v", status, http.StatusOK)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	c := config{}
	c.Server.Addr = ":8080"
	c.Server.WriteTimeout = time.Second * 15
	c.Server.ReadTimeout = time.Second * 15
	c.Server.IdleTimeout = time.Second * 60
	c.Database.RedisMaster = "redis-master:6379"
	c.Database.RedisSlave = "redis-slave:6379"
	cfg := loadConfig("toto")
	if c != cfg {
		t.Errorf("unexpected conf, got %v, want %v", cfg, c)
	}
}
