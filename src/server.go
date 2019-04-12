package main

import (
	"github.com/gorilla/mux"
	"fmt"
	"context"
	"net/http"
	"log"
	"time"
	"os"
	"io"	
	"strconv"
	//"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/mongo"
        "go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
        useHttps bool
	port string
	dbhost string
	dbport string
	psk string
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

type AVote struct {
	Msgid int    `json:"msgid"`
	Uuid string  `json:"uuid"`
	Vote int     `json:"vote"`
}
  
type AllVotes struct {
	Msgid int      `json:"msgid"`
	Upvotes int    `json:"upvotes"`
	Downvotes int  `json:"downvotes"`
}
 
type DBConnectionContext struct {
	client *mongo.Client
	imaginevotes *mongo.Collection
	imaginevotestats *mongo.Collection
	realvotes *mongo.Collection
	realvotestats *mongo.Collection
	virtualvotes *mongo.Collection
	virtualvotestats *mongo.Collection
}

var votes []VoteStats


func GetConfig() Config {
        newConf := Config{}
        newConf.useHttps = false
	newConf.port = "9911"
	newConf.dbhost = "localhost"
	newConf.dbport = "27017"
	newConf.psk = "31416"
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
	if(os.Getenv("PSK") != "") {
                newConf.port = os.Getenv("PSK")
        }
        return newConf
}

func connectToDB() (*mongo.Client, error) {
        clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
        return mongo.Connect(context.TODO(), clientOptions)
}

func registerRealVote(conn *DBConnectionContext, schoolid string, amount int) (error) {
	_, err := conn.realvotes.InsertOne(context.TODO(), bson.D{{"schoolid",schoolid},{"amount",amount}})
	return err
}

func registerVirtualVote(conn *DBConnectionContext, groupid string, amount int) (error) {
	_, err := conn.virtualvotes.InsertOne(context.TODO(), bson.D{{"groupid",groupid},{"amount",amount}})
	return err
}

func registerImagineVote(conn *DBConnectionContext, msgid int, uuid string, vote int) {
        if vote > 1 {
                 vote = 1
        }
        if vote < -1 {
                 vote = -1
        }
        var result AVote
        err := conn.imaginevotes.FindOne(context.TODO(), bson.D{{"uuid", uuid},{"msgid", msgid}}).Decode(&result)
        if err!=nil {
                newVote := AVote{msgid, uuid, vote}
                _, err := conn.imaginevotes.InsertOne(context.TODO(), newVote)
                if err!=nil {
                        log.Fatal(err)
                }
                var allVoteResult AllVotes
                err = conn.imaginevotestats.FindOne(context.TODO(), bson.D{{"msgid", msgid}}).Decode(&allVoteResult)
                if err!=nil || allVoteResult.Msgid!=msgid {
                        fmt.Println("No previous record found, creating one -> ", allVoteResult)
                        var newStat AllVotes
                        newStat.Msgid = msgid
                        if vote > 0 {
                                newStat.Upvotes = 1
                                newStat.Downvotes = 0
                        } else {
                                newStat.Upvotes = 0
                                newStat.Downvotes = 1
                        }
                        fmt.Println("Trying to insert: ", newStat)
                        _, err = conn.imaginevotestats.InsertOne(context.TODO(), newStat)
                        if err!=nil {
                                log.Fatal(err)
                        }
		} else {
			fmt.Println("Previous record found: ", allVoteResult)
			var action bson.D
			if vote > 0 {
				action = bson.D{{"$inc", bson.D{{"upvotes", 1}}}}
			}
			if vote < 0 {
				action = bson.D{{"$inc", bson.D{{"downvotes", 1}}}}
			}
			_, err = conn.imaginevotestats.UpdateOne(context.TODO(), bson.D{{"msgid", msgid}}, action)
			if err!=nil {
				log.Fatal(err)
			}
		}
	}

}

func validRange(amount int) (bool) {
	if(amount < 0) {
		return false
	}
	if(amount >= 10000) {
		return false
	}
	return true
}

func HCHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "{\"alive\":true}")
}

func JSONResponseFromString(w http.ResponseWriter, res string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, res)
}

func RealPostVote(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	schoolid := r.FormValue("schoolid")
	amount, err := strconv.Atoi(r.FormValue("amount"))
	if(err==nil && validRange(amount)) {
		_ = registerRealVote(&dbConnectionContext, schoolid, amount)
		JSONResponseFromString(w, "{\"result\":\"OK\"}")
	} else {
		JSONResponseFromString(w, "{\"result\":\"Amount Error\"}")
	}
	
}

func RealGetVotes(w http.ResponseWriter, r *http.Request) {

}

func VirtualPostVote(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	groupid := r.FormValue("groupid")
	amount, err := strconv.Atoi(r.FormValue("amount"))
	if(err == nil && validRange(amount)) {
		_ = registerVirtualVote(&dbConnectionContext, groupid, amount)
		JSONResponseFromString(w, "{\"result\":\"OK\"}")
	} else {
		JSONResponseFromString(w, "{\"result\":\"Amount Error\"}")
	}
	
}

func VirtualGetVotes(w http.ResponseWriter, r *http.Request) {

}

func ImaginePostVote(w http.ResponseWriter, r *http.Request) {
	//params := mux.Vars(s)
}

func ImagineGetVotes(w http.ResponseWriter, r *http.Request) {
	//params := mux.Vars(r)
	//for _, item := range votes {
	//	if item.Msgid == params["msgid"] {
	//		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//		w.WriteHeader(http.StatusOK)
	//		json.NewEncoder(w).Encode(votes)
	//	}
	//}
	
}

var globalConfig Config

var dbConnectionContext DBConnectionContext

func withPSKCheck(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Checking PSK...")
		next.ServeHTTP(w, r)
	}
}

func main() {

	client, err := connectToDB()
        err = client.Ping(context.TODO(), nil)

        if err != nil {
                log.Fatal(err)
        }

	fmt.Println("Connected to MongoDB!")
	
	dbConnectionContext.imaginevotes = client.Database("imagine").Collection("imaginevotes")
	dbConnectionContext.imaginevotestats = client.Database("imagine").Collection("imaginevotestats")
	dbConnectionContext.realvotes = client.Database("imagine").Collection("realbottlevotes")
	dbConnectionContext.virtualvotes = client.Database("imagine").Collection("virtualbottlevotes")

	router := mux.NewRouter()
	globalConfig = GetConfig()
	fmt.Println(globalConfig)	

	router.HandleFunc("/healthcheck", withPSKCheck(HCHandler)).Methods("GET")
	router.HandleFunc("/imagine/votes", withPSKCheck(ImaginePostVote)).Methods("POST")
	router.HandleFunc("/imagine/votes", withPSKCheck(ImagineGetVotes)).Methods("GET")
	router.HandleFunc("/real/votes", withPSKCheck(RealPostVote)).Methods("POST")
	router.HandleFunc("/real/votes", withPSKCheck(RealGetVotes)).Methods("GET")
	router.HandleFunc("/virtual/votes", withPSKCheck(VirtualPostVote)).Methods("POST")
	router.HandleFunc("/virtual/votes", withPSKCheck(VirtualGetVotes)).Methods("GET")

	srv := &http.Server {
		Handler: router,
		Addr:	 "0.0.0.0:"+globalConfig.port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println("Imagine backend server started at port " + globalConfig.port)
	log.Fatal(srv.ListenAndServe())

}
