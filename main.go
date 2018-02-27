package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

/*
* [TODO]:
* - Connect to Mongo DB
*		1. Install mgo package: go get gopkg.in/mgo.v2
*		2. Add adapter Pattern Type Helper method
* -Inserts Data
* - Reads Data
 */

type Adapter func(http.Handler) http.Handler

type comment struct {
	ID     bson.ObjectId `json:"id" bson:"_id"`
	Author string        `json:"author" bson:"author"`
	Text   string        `json:"text" bson:"text"`
	When   time.Time     `json:"when" bson:"when"`
}

func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	/* this will help us run code before and/or after HTTP requests come to the
	 * API*/

	for _, adapter := range adapters {
		h = adapter(h)
	}

	return h
}

func withDB(db *mgo.Session) Adapter {
	//Return the Adapter
	return func(h http.Handler) http.Handler {
		// the adapter (when called) shpuld return a new handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//copy the database session
			dbsession := db.Copy()
			defer dbsession.Close() // clean up

			//save it in the mux context
			context.Set(r, "database", dbsession)

			// pass execution to the original handler
			h.ServeHTTP(w, r)

		})
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleRead(w, r)
	case "POST":
		handleInsert(w, r)
	default:
		http.Error(w, "Not Supported", http.StatusMethodNotAllowed)
	}
}

func handleInsert(w http.ResponseWriter, r *http.Request) {
	db := context.Get(r, "database").(*mgo.Session)

	//decorate the request body
	var c comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//give the comment a unique ID and set the time
	c.ID = bson.NewObjectId()
	c.When = time.Now()

	//insert into the database
	if err := db.DB("commentsapp").C("comments").Insert(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//redirect to the comment
	http.Redirect(w, r, "/comments/"+c.ID.Hex(), http.StatusTemporaryRedirect)
}

func main() {

	//connect to the database
	db, err := mgo.Dial("mongodb://bd-sbxcardart01:mRrSShNPTDvMhKEaUr3quO8BCBHAA025xEam7MafIUBmEOM77abHoFYIaYPBTgrjYvg445Pnnf6nB4DAPaXa7w==@bd-sbxcardart01.documents.azure.com:10255/?ssl=true&replicaSet=globaldb")
	if err != nil {
		log.Fatal("cannot dial mongoñongo", err)
	}

	defer db.Close() //cleanup when done

	//Adapt our handle function withDB
	h := Adapt(http.HandlerFunc(handle), withDB(db))

	//add the handler http
	http.Handle("/comments", context.ClearHandler(h))

	//start the server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

}
