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

/*
// v1 of this server only sends out a slice of readings
type sourceStruct struct {
	Readings []float32 // temp sensor readings
}
*/

// now in version 2, imagine that the server-side implemention (i.e. this program) decided it
// was better to have the front-end display filtered readings (instead of noisy raw readings).
// However, imagine there was a requiment that this updated server must be backward-compatible,
// despite the fact that the actual data that the server is sending out has been changed.
// Additionally, imagine that in newer version of the front end, we wanted to display higher-resolution (more precise)
// versions of our readings. Therefore, we decided to send the values out as float64's. This hypothetical scenario
// let's us illustrate here how the Schemer library itself will allow decoding of float64's into float32's
type sourceStruct struct {
	Header string // example of this is new string-based value that was added in version 2 of this system
	// in newer version of the front end, we might now want to include both the raw readings and the filtered version
	RawReadings []float64
	// however, for backwards compatibility...
	// notice the struct tag here: we want the old front end to just
	// see these new filtered values as its "readings" w/o even knowing
	// that anything changed!
	FilteredReadings []float64 `schemer:"readings"`
}

var mu sync.Mutex
var structToEncode = sourceStruct{}
var writerSchema = schemer.SchemaOf(&structToEncode)
var binaryWriterSchema []byte

// this is original version
/*
func asyncUpdate() {

	mu.Lock()
	defer mu.Unlock()

	numFloats := rand.Intn(10)
	structToEncode.Readings = make([]float32, numFloats)

	for i := 0; i < numFloats; i++ {
		structToEncode.Readings[i] = float32(rand.Intn(10000000))
	}

}
*/

func asyncUpdate() {

	mu.Lock()
	defer mu.Unlock()

	// in version 2.0, imagine for some reason we want to send out a string-based header now
	// (for the header in this version, this example just uses some famous text from Abraham Lincoln)
	// Version 1.0 of the client will just ignore this [because when it written this header didn't exist]
	randomStrings := []string{
		"Four score and seven years ago",
		"our fathers brought forth on this continent,",
		"a new nation,",
		"conceived in Liberty,",
		"and dedicated to the proposition that all men",
		"are created equal.",
		"Now we are engaged in a great civil war,",
		"testing whether that nation,",
		"or any nation so conceived and so dedicated,",
		"can long endure.",
		"We are met on a great battle-field of that war.",
		"We have come to dedicate a portion of that field,",
		"as a final resting place",
		"for those who here gave their lives that",
		"that nation might live.",
		"It is altogether fitting and",
		"proper that we should do this.",
	}

	randomIndex := rand.Intn(len(randomStrings))
	structToEncode.Header = randomStrings[randomIndex]

	// now in version 2.0 of this server, imagine we want to send over
	// both the raw readings and the filtered readings

	numFloats := rand.Intn(10)
	structToEncode.RawReadings = make([]float64, numFloats)

	for i := 0; i < numFloats; i++ {
		structToEncode.RawReadings[i] = float64(rand.Intn(10000000))
	}

	// put a simple filter on the values
	structToEncode.FilteredReadings = make([]float64, numFloats)
	smoothingFactor := 0.5

	var workingAverage float64 = 0.0
	for i := 0; i < numFloats; i++ {
		newValue := structToEncode.RawReadings[i]
		workingAverage = (newValue * smoothingFactor) + (workingAverage * (1.0 - smoothingFactor))
		structToEncode.FilteredReadings[i] = workingAverage
	}

}

func getSchemaHanlder() http.HandlerFunc {
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

func getDataHanlder() http.HandlerFunc {
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
Additionally, this example also illustrates how the server-side component (or a client-server system) can evolve
what it is sending, while still maintaing backwards compatibility with an older version of the client (that still
expects the original data.)   
	`
	fmt.Println(s)

}

func main() {
	binaryWriterSchema = writerSchema.MarshalSchemer()

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	rand.Seed(time.Now().UnixNano())

	// constantly write out new data
	go asyncUpdate()

	// setup our endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/get-schema/", getSchemaHanlder())
	mux.HandleFunc("/get-data/", getDataHanlder())

	printIntro()

	log.Println("example server listing on port:", port)
	log.Println("endpont 1: /get-schema/")
	log.Println("endpont 2: /get-data/")

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
