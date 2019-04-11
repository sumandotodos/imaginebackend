package main

import (
	"github.com/gorilla/mux"
	"fmt"
	"net/http"
	"log"
	"time"
	"os"
	"io"	
	"encoding/json"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
)

type Config struct {
        useHttps bool
	port string
	dbhost string
	dbport string
}

type Vote struct {
	uuid string
	msgid int
	vote int
}

type VoteStats struct {
	Msgid int     `json:"msgid"`
	Upvotes int   `json:"upvotes"`
	Downvotes int `json:"downvotes"`
}

var votes []VoteStats

func GetConfig() Config {
        newConf := Config{}
        newConf.useHttps = false
	newConf.port = "9911"
	newConfig.dbhost = "localhost"
	newConfig.dbport = "27017"
	if(os.Getenv("USE_HTTPS") == "YES") {
                newConf.useHttps = true
        }
        if(os.Getenv("PORT") != "") {
                newConf.port = os.Getenv("PORT")
	}
	if(os.Getenv("DBPORT") != "") {
                newConf.dbport = os.Getenv("DBPORT")
	}
	if(os.Getenv("DBHOST") != "") {
                newConf.port = os.Getenv("DBHOST")
        }
        return newConf
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "{\"alive\":true}")
}

func PostVote(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(s)

}

func GetVotes(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for _, item := range votes {
		if item.Msgid == params["msgid"] {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(votes)
		}
	}
	
}

func connectToDB() {
	url := "mongodb://" + globalConfig.dbhost + ":" + globalConfig.dbport
}

var globalConfig Config

func main() {
	router := mux.NewRouter()
	globalConfig = GetConfig()
	fmt.Println(newConfig)	

	router.HandleFunc("/healthcheck", TestHandler).Methods("GET")
	router.HandleFunc("/votes", PostVote).Methods("POST")
	router.HandleFunc("/votes", GetVotes).Methods("GET")

	srv := &http.Server {
		Handler: router,
		Addr:	 "0.0.0.0:"+newConfig.port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println("Imagine backend server started at port " + newConfig.port)
	log.Fatal(srv.ListenAndServe())

}
