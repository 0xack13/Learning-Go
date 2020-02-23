package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

//type server struct{}

//type comes after var name in golang
//For server to implement http.Handler interface (and be used as that type), need to define ServeHTTP method
/*
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Print(r.URL)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "hello world"}`))
}
*/

// All of this thread safety likely isn't necessary as I'm not running a multithreaded server, right?
// This is wrong!
// Instead of a mutex, use channels: https://eli.thegreenplace.net/2019/on-concurrency-in-go-http-servers/

//don't capitalize unless you want to export
/*
type safeCount struct {
	v   map[string]int
	mux sync.Mutex
}

func (c *safeCount) inc(key string) {
	c.mux.Lock()
	c.v[key]++
	c.mux.Unlock()
}
*/

func timeFunc(ticker *time.Ticker, mainCounter *appState) {
	// empty for is a while
	for {
		select {
		case <-ticker.C:
			fmt.Println("Timer Called")
			fmt.Println(mainCounter.val1)
			fmt.Println(mainCounter.val2)
			fmt.Fprintf(os.Stderr, "err test")
		}
	}
}

// trying to manage shared state w/ channels
// handlers send over shared channel
// and provide a channel with which they will receive
type appState struct {
	val1       int
	val2       int
	csvRecords [][]string
	status     bool
}

type cmdType int

const (
	increment     cmdType = 0
	updateRecords cmdType = 1
	getStatus     cmdType = 2
)

type appStateCmd struct {
	cmd    cmdType
	record []string
	//lets handler pass in their receiver
	jsonReceiver chan []byte
}

type handlerState struct {
	appStateChan chan<- appStateCmd
}

//use field tags to get lowercase
type statusResponse struct {
	Status string `json:"status"`
	Ts     string `json:"ts"`
	Msg    string `json:"msg"`
}

// counter manager will:
// - initialize state
// - start a goroutine
// - return a channel
func startCounterManager(appState *appState) chan<- appStateCmd {

	appStateChan := make(chan appStateCmd)

	// start anonymous goroutine
	go func() {
		// has access to appStateChan via closure
		for cmd := range appStateChan {
			switch cmd.cmd {
			case increment:
				fmt.Println("Receiving on counter channel")
				appState.val1++
				appState.val2 += 10
			case getStatus:
				t := time.Now()
				ts := t.Format(time.RFC3339)
				if appState.status {
					fmt.Println("sending success status")
					resp := &statusResponse{Status: "ok", Msg: "", Ts: ts}
					respBytes, _ := json.Marshal(resp)
					cmd.jsonReceiver <- respBytes
				} else {
					fmt.Println("sending failure status")
					resp := &statusResponse{Status: "error", Msg: "failed to load csv", Ts: ts}
					respBytes, _ := json.Marshal(resp)
					cmd.jsonReceiver <- respBytes
				}
			}
		}
	}()
	return appStateChan
}

// global until i figure the right approach: https://stackoverflow.com/questions/26211954/how-do-i-pass-arguments-to-my-handler
//var appStateChan = startCounterManager()

// straight func, no method signature
func (state *handlerState) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "get called"}`))
		fmt.Println("GET handler home called")
		handlerChan := make(chan []byte)
		msg := appStateCmd{cmd: increment, jsonReceiver: handlerChan}
		state.appStateChan <- msg
	case "POST":
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "post called"}`))
	case "PUT":
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"message": "put called"}`))
	case "DELETE":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delete called"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}
}

func (state *handlerState) favicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "favicon get called"}`))
		fmt.Println("favicon")
	case "POST":
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "post called"}`))
	case "PUT":
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"message": "put called"}`))
	case "DELETE":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delete called"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}
}

func (state *handlerState) test(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test get called"}`))
		fmt.Println("GET handler test called")
		handlerChan := make(chan []byte)
		msg := appStateCmd{cmd: increment, jsonReceiver: handlerChan}
		state.appStateChan <- msg
	case "POST":
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "post called"}`))
	case "PUT":
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"message": "put called"}`))
	case "DELETE":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delete called"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}
}

func (state *handlerState) status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		fmt.Println("GET handler status called")
		handlerChan := make(chan []byte)
		msg := appStateCmd{cmd: getStatus, jsonReceiver: handlerChan}
		state.appStateChan <- msg
		resp := <-handlerChan
		fmt.Printf("resp is: %s", resp)
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	case "POST":
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "post called"}`))
	case "PUT":
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"message": "put called"}`))
	case "DELETE":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delete called"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}
}

func main() {
	// reference to a server struct??
	//s := &server{}

	//http.Handle("/", s)

	csvError := false
	csvfile, err := os.Open("./winemag-data-130k-v2.csv")
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
		csvError = true
	}
	r := csv.NewReader(csvfile)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatalln("Couldn't read the csv file", err)
		csvError = true
	}

	appState := &appState{val1: 0, val2: 0, csvRecords: records, status: !csvError}
	fmt.Println("records:", len(records))

	ticker := time.NewTicker(time.Second * 5)

	//timer func doesn't need thread-safe access to appState
	go timeFunc(ticker, appState)

	// this is so wild, i need to know what's happening here
	handlerState := handlerState{
		appStateChan: startCounterManager(appState)}

	// investigate better approaches to giving these funcs access to appStateChan
	http.HandleFunc("/", handlerState.home) // also serves as default handler
	http.HandleFunc("/favicon.ico", handlerState.favicon)
	http.HandleFunc("/test", handlerState.test)
	http.HandleFunc("/status", handlerState.status)

	defer ticker.Stop() // are defers necessary in main?
	defer close(handlerState.appStateChan)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
