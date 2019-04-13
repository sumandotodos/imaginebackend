package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	//"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	useHttps bool
	port     string
	dbhost   string
	dbport   string
	psk      string
}

type Vote struct {
	uuid  string
	msgid int
	vote  int
}

type UniqueIncrementIndex struct {
	Uuid  int `json:"uuid"`
	Index int `json:"index"`
}

type VoteStats struct {
	Msgid     int `json:"msgid"`
	Upvotes   int `json:"upvotes"`
	Downvotes int `json:"downvotes"`
}

type RealVote struct {
	Schoolid string `json:"schoolid"`
	Amount   int    `json:"amount"`
}

type VirtualVote struct {
	Groupid string `json:"groupip"`
	Amount  int    `json:"amount"`
}

type AVote struct {
	Msgid int    `json:"msgid"`
	Uuid  string `json:"uuid"`
	Vote  int    `json:"vote"`
}

type AllVotes struct {
	Msgid     int `json:"msgid"`
	Upvotes   int `json:"upvotes"`
	Downvotes int `json:"downvotes"`
}

type DBConnectionContext struct {
	client           *mongo.Client
	imaginevotes     *mongo.Collection
	imaginevotestats *mongo.Collection
	realvotes        *mongo.Collection
	realvotestats    *mongo.Collection
	virtualvotes     *mongo.Collection
	virtualvotestats *mongo.Collection
	uniqueindex      *mongo.Collection
}

var votes []VoteStats

func GetConfig() Config {

	newConf := Config{}
	newConf.useHttps = false
	newConf.port = "9911"
	newConf.dbhost = "localhost"
	newConf.dbport = "27017"
	newConf.psk = "31416"
	if os.Getenv("USE_HTTPS") == "YES" {
		newConf.useHttps = true
	}
	if os.Getenv("PORT") != "" {
		newConf.port = os.Getenv("PORT")
	}
	if os.Getenv("DBPORT") != "" {
		newConf.dbport = os.Getenv("DBPORT")
	}
	if os.Getenv("DBHOST") != "" {
		newConf.dbhost = os.Getenv("DBHOST")
	}
	if os.Getenv("PSK") != "" {
		newConf.psk = os.Getenv("PSK")
	}
	return newConf

}

func connectToDB(conf Config) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI("mongodb://" + conf.dbhost + ":" + conf.dbport)
	return mongo.Connect(context.TODO(), clientOptions)
}

func registerRealVote(conn *DBConnectionContext, schoolid string, amount int) error {
	_, err := conn.realvotes.InsertOne(context.TODO(), bson.D{{"schoolid", schoolid}, {"amount", amount}})
	return err
}

func registerVirtualVote(conn *DBConnectionContext, groupid string, amount int) error {
	_, err := conn.virtualvotes.InsertOne(context.TODO(), bson.D{{"groupid", groupid}, {"amount", amount}})
	return err
}

func getAndIncrementIndex(conn *DBConnectionContext) (int, error) {
	var index UniqueIncrementIndex
	err := conn.uniqueindex.FindOne(context.TODO(), bson.D{{"uuid", 0}}).Decode(&index)
	if err != nil {
		fmt.Println("Something funky getting uniqueindex, creating new index")
		_, _ = conn.uniqueindex.InsertOne(context.TODO(), bson.D{{"uuid", 0}, {"index", 0}})
	}
	index.Index++
	_, err = conn.uniqueindex.UpdateOne(context.TODO(), bson.D{{"uuid", 0}}, bson.D{{"$set", bson.D{{"index", index.Index}}}})
	if err != nil {
		fmt.Println("Error en updateone")
		fmt.Println(err)
	}
	return index.Index - 1, err
}

func getRealVotes(conn *DBConnectionContext, schoolid string) ([]int, error) {
	cursor, _ := conn.realvotes.Find(context.TODO(), bson.D{{"schoolid", schoolid}})
	defer cursor.Close(context.TODO())
	result := make([]int, 0)
	for cursor.Next(context.TODO()) {
		var realVote RealVote
		cursor.Decode(&realVote)
		result = append(result, realVote.Amount)
	}
	return result, nil
}

func getImagineVotes(conn *DBConnectionContext, msgid int) (int, int) {
	var votes AllVotes
	err := conn.imaginevotestats.FindOne(context.TODO(), bson.D{{"msgid", msgid}}).Decode(&votes)
	if err == nil {
		if votes.Upvotes == 0 && votes.Downvotes == 0 {
			fmt.Println("1")
			return 0.0, 0.0
		} else {
			fmt.Println("2")
			fractionUp := float32(votes.Upvotes) / (float32(votes.Upvotes) + float32(votes.Downvotes))
			percentUp := fractionUp * 100.0
			return int(percentUp), 100 - int(percentUp)
		}
	} else {
		fmt.Println("3")
		fmt.Println(err)
		return 0.0, 0.0
	}
}

func registerImagineVote(conn *DBConnectionContext, msgid int, vote int) float32 {

	if vote > 1 {
		vote = 1
	}
	if vote < -1 {
		vote = -1
	}

	var votes AllVotes
	deltaup := 0
	deltadown := 0

	err := conn.imaginevotestats.FindOne(context.TODO(), bson.D{{"msgid", msgid}}).Decode(&votes)
	if err != nil {
		fmt.Println("No collection found, creating one...")
		fmt.Println(err)
		_, _ = conn.imaginevotestats.InsertOne(context.TODO(), bson.D{{"msgid", msgid}, {"upvotes", 0}, {"downvotes", 0}})
	}

	action := bson.D{{}}
	if vote > 0 {
		action = bson.D{{"$inc", bson.D{{"upvotes", 1}}}}
		deltaup = 1
	}
	if vote < 0 {
		action = bson.D{{"$inc", bson.D{{"downvotes", 1}}}}
		deltadown = 1
	}

	_, err = conn.imaginevotestats.UpdateOne(context.TODO(), bson.D{{"msgid", msgid}}, action)
	if err != nil {
		log.Fatal(err)
	}

	return float32(votes.Upvotes+deltaup) / (float32(votes.Downvotes+deltadown) + float32(votes.Upvotes+deltaup))

}

func validRange(amount int) bool {
	if amount < 0 {
		return false
	}
	if amount >= 10000 {
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

func JSONResponseFromStringAndCode(w http.ResponseWriter, res string, status int) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	io.WriteString(w, res)
}

func RealPostVote(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	schoolid := r.FormValue("schoolid")
	amount, err := strconv.Atoi(r.FormValue("amount"))
	if err == nil && validRange(amount) {
		_ = registerRealVote(&dbConnectionContext, schoolid, amount)
		JSONResponseFromString(w, "{\"result\":\"OK\"}")
	} else {
		JSONResponseFromString(w, "{\"result\":\"Amount Error\"}")
	}

}

func RealGetVotes(w http.ResponseWriter, r *http.Request) {
	var amounts []int
	r.ParseForm()
	if r.FormValue("schoolid") == "" {
		JSONResponseFromString(w, "{\"result\":\"schoolid not specified\"}")
	} else {
		amounts, _ = getRealVotes(&dbConnectionContext, r.FormValue("schoolid"))
		listOfNumbers := "[ "
		for index, value := range amounts {
			if index == 0 {
				listOfNumbers = listOfNumbers + strconv.Itoa(value)
			} else {
				listOfNumbers = listOfNumbers + ", " + strconv.Itoa(value)
			}
		}
		listOfNumbers = listOfNumbers + "]"
		JSONResponseFromString(w, "{\"result\":"+listOfNumbers+"}")
	}
}

func VirtualPostVote(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	groupid := r.FormValue("groupid")
	amount, err := strconv.Atoi(r.FormValue("amount"))
	if err == nil && validRange(amount) {
		_ = registerVirtualVote(&dbConnectionContext, groupid, amount)
		JSONResponseFromString(w, "{\"result\":\"OK\"}")
	} else {
		JSONResponseFromString(w, "{\"result\":\"Amount Error\"}")
	}

}

func VirtualGetVotes(w http.ResponseWriter, r *http.Request) {

}

func ImaginePostVote(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	msgid, _ := strconv.Atoi(r.FormValue("msgid"))
	vote, _ := strconv.Atoi(r.FormValue("vote"))
	if vote == 0 {
		vote = 1
	}
	fractionUp := registerImagineVote(&dbConnectionContext, msgid, vote)
	fraction := fractionUp
	if vote < 0 {
		fraction = (1.0 - fractionUp)
	}
	JSONResponseFromString(w, "{\"same-percentage\":"+strconv.Itoa(int(fraction*100.0))+"}")
}

func GetUniqueIndex(w http.ResponseWriter, r *http.Request) {
	index, _ := getAndIncrementIndex(&dbConnectionContext)
	JSONResponseFromString(w, "{\"uniqueindex\":"+strconv.Itoa(index)+"}")
}

func ImagineGetVotes(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	msgid, _ := strconv.Atoi(r.FormValue("msgid"))
	pUp, pDown := getImagineVotes(&dbConnectionContext, msgid)
	JSONResponseFromString(w, "{\"percent-up\":"+strconv.Itoa(pUp)+", \"percent-down\":"+strconv.Itoa(pDown)+"}")

}

var globalConfig Config

var dbConnectionContext DBConnectionContext

func withPSKCheck(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		psk := r.Header.Get("psk")
		if psk == globalConfig.psk {
			next.ServeHTTP(w, r)
		} else {
			JSONResponseFromStringAndCode(w, "{\"result\":\"forbidden\"}", 403)
		}
	}
}

func main() {

	globalConfig = GetConfig()
	fmt.Println(globalConfig)

	client, err := connectToDB(globalConfig)
	err = client.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	dbConnectionContext.imaginevotes = client.Database("imagine").Collection("imaginevotes")
	dbConnectionContext.imaginevotestats = client.Database("imagine").Collection("imaginevotestats")
	dbConnectionContext.realvotes = client.Database("imagine").Collection("realbottlevotes")
	dbConnectionContext.virtualvotes = client.Database("imagine").Collection("virtualbottlevotes")
	dbConnectionContext.uniqueindex = client.Database("imagine").Collection("uniqueindex")

	router := mux.NewRouter()

	router.HandleFunc("/healthcheck", HCHandler).Methods("GET")
	router.HandleFunc("/imagine/votes", withPSKCheck(ImaginePostVote)).Methods("POST")
	router.HandleFunc("/imagine/votes", withPSKCheck(ImagineGetVotes)).Methods("GET")
	router.HandleFunc("/real/votes", withPSKCheck(RealPostVote)).Methods("POST")
	router.HandleFunc("/real/votes", RealGetVotes).Methods("GET")
	router.HandleFunc("/virtual/votes", withPSKCheck(VirtualPostVote)).Methods("POST")
	router.HandleFunc("/virtual/votes", VirtualGetVotes).Methods("GET")
	router.HandleFunc("/uniqueindex", withPSKCheck(GetUniqueIndex)).Methods("GET")

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + globalConfig.port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println("Imagine backend server started at port " + globalConfig.port)
	log.Fatal(srv.ListenAndServe())

}
