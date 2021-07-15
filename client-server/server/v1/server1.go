package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bminer/schemer"
)

const DefaultPort = "8080"

// v1 of this server only sends out a slice of readings
type sourceStruct struct {
	Readings []float32 // temp sensor readings
}

var mu sync.Mutex
var structToEncode = sourceStruct{}
var writerSchema = schemer.SchemaOf(&structToEncode)
var binaryWriterSchema []byte

func asyncUpdate() {

	mu.Lock()
	defer mu.Unlock()

	numFloats := rand.Intn(10)
	structToEncode.Readings = make([]float32, numFloats)

	for i := 0; i < numFloats; i++ {
		structToEncode.Readings[i] = float32(rand.Intn(10000000))
	}

}

func getSchemaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		if req.Method != http.MethodGet {
			http.Error(w, "Invalid Invocation", http.StatusNotFound)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")

		buf := bytes.NewBuffer(binaryWriterSchema)
		_, err := w.Write(buf.Bytes())

		if err != nil {
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			log.Println("i/o error: " + err.Error())
			return
		}

		log.Printf("successfully returned binary schema")
	}
}

func getDataHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		if req.Method != http.MethodGet {
			http.Error(w, "Invalid Invocation", http.StatusNotFound)
			return
		}

		mu.Lock()

		var encodedData bytes.Buffer
		err := writerSchema.Encode(&encodedData, structToEncode)
		if err != nil {
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			defer mu.Unlock()
			return
		}

		mu.Unlock()

		n, err := w.Write(encodedData.Bytes())
		log.Printf("%d bytes written ", n)

		if err != nil {
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			log.Println("i/o error: " + err.Error())
			return
		}

		log.Printf("successfully returned binary data")
	}
}

func printIntro() {

	s := `
This is an example of a server  that listens (for incoming HTTP connections) either on port 8080 (the default),
or some other port specified in the environment called PORT. 
This example is designed to illustrate how to create a GO-based server that uses the Schemer to send binary data
over the wire. 
Additionally, this example also illustrates how the server-side component (of a client-server system) can evolve
what it is sending, while still maintaing backwards compatibility with an older version of the client (that still
expects the original data.)   
	`
	fmt.Println(s)

}

func main() {
	binaryWriterSchema, _ = writerSchema.MarshalJSON()

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	rand.Seed(time.Now().UnixNano())

	// constantly write out new data
	go asyncUpdate()

	// setup our endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/get-schema/", getSchemaHandler())
	mux.HandleFunc("/get-data/", getDataHandler())

	printIntro()

	log.Println("example server listing on port:", port)
	log.Println("endpont 1: /get-schema/")
	log.Println("endpont 2: /get-data/")

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
