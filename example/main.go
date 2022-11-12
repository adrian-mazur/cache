package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/adrian-mazur/cache"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

const maxRetries = 5
const retryWaitDuration = 100 * time.Millisecond
const lockExpiration = 5 * time.Second

var db *sql.DB
var userCache *cache.Cache[User]
var ctx = context.Background()

type User struct {
	Id        int
	FirstName string
	LastName  string
}

func (u User) SerializeToString() (string, error) {
	bytes, err := json.Marshal(u)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (u User) DeserializeFromString(str string) (User, error) {
	var result User
	err := json.Unmarshal([]byte(str), &result)
	return result, err
}

func getUserById(id int) (User, error) {
	getFromDatabaseFunc := func() (User, error) {
		var user User
		rows, err := db.Query("select id, first_name, last_name from users where id = ?", id)
		if err != nil {
			return user, err
		}
		if !rows.Next() {
			return user, nil
		}
		if err := rows.Scan(&user.Id, &user.FirstName, &user.LastName); err != nil {
			return user, err
		}
		return user, nil
	}
	return userCache.GetOrSetIfDoesNotExist(ctx, strconv.Itoa(id), maxRetries, retryWaitDuration, getFromDatabaseFunc)
}

func userHandler(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Print(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := getUserById(userId)
	if err != nil {
		log.Print(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	if user.Id == 0 {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	rw.Header().Add("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(rw).Encode(user); err != nil {
		log.Print(err)
	}
}

func setUpDb() error {
	_, err := db.Exec(`
		create table users(id integer primary key, first_name string, last_name string);
		insert into users (first_name, last_name) values ("John", "Doe");
		insert into users (first_name, last_name) values ("Jan", "Kowalski");
		insert into users (first_name, last_name) values ("Jan", "Nowak");
	`)
	return err
}

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:3000"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	var err error
	db, err = sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	if err := setUpDb(); err != nil {
		log.Fatal(err)
	}

	redis := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	userCache = cache.NewCache[User](cache.NewRedisCacheStorage(redis, "users"), cache.NewRedisLocker(redis, "users-lock", lockExpiration))

	r := mux.NewRouter()
	r.HandleFunc("/api/users/{id:[0-9]+}", userHandler)
	http.Handle("/", r)

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
