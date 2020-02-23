package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

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

type headerName int

const (
	id                  headerName = 0
	country             headerName = 1
	description         headerName = 2
	designation         headerName = 3
	points              headerName = 4
	price               headerName = 5
	province            headerName = 6
	region1             headerName = 7
	region2             headerName = 8
	tasterName          headerName = 9
	tasterTwitterHandle headerName = 10
	title               headerName = 11
	variety             headerName = 12
	winery              headerName = 13
)

type appState struct {
	val1       int
	val2       int
	csvRecords [][]string
	status     bool
}

type cmdType int

const (
	increment   cmdType = 0
	putWine     cmdType = 1
	getStatus   cmdType = 2
	getWines    cmdType = 3
	getWineByID cmdType = 4
)

type appStateCmd struct {
	cmd        cmdType
	wineID     int
	putRecord  []byte
	startIndex int
	pageCount  int
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

type wineResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type wineListResponse struct {
	Wines []wineResponse `json:"wines"`
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
			case getWines:
				if (cmd.startIndex < 0) || (cmd.pageCount < 0) {
					wineList := make([]wineResponse, len(appState.csvRecords))
					for i, record := range appState.csvRecords {
						newRecord := wineResponse{ID: record[id], Title: record[title]}
						wineList[i] = newRecord
					}
					resp := wineListResponse{Wines: wineList}
					respBytes, _ := json.Marshal(resp)
					cmd.jsonReceiver <- respBytes
				} else {
					if cmd.startIndex >= len(appState.csvRecords) {
						//error condition
						cmd.jsonReceiver <- []byte(`{"error":"start index larger than record size"}`)
					} else if (cmd.startIndex + cmd.pageCount) >= len(appState.csvRecords) {
						// another error condition
						cmd.jsonReceiver <- []byte(`{"error":"start index + count larger than record size"}`)
					} else {
						wineListSlice := appState.csvRecords[cmd.startIndex : cmd.startIndex+cmd.pageCount]
						wineList := make([]wineResponse, len(wineListSlice))
						for i, record := range wineListSlice {
							newRecord := wineResponse{ID: record[id], Title: record[title]}
							wineList[i] = newRecord
						}
						resp := wineListResponse{Wines: wineList}
						respBytes, _ := json.Marshal(resp)
						cmd.jsonReceiver <- respBytes
					}

				}
			case getWineByID:
				//
			case putWine:
				//
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
		w.Write([]byte(`{"message": "default handler called"}`))
		fmt.Println("default handler home called - will not increment")
		//handlerChan := make(chan []byte)
		//msg := appStateCmd{cmd: increment, jsonReceiver: handlerChan}
		//state.appStateChan <- msg
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

func (state *handlerState) getWines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		startStr := r.URL.Query().Get("start")
		count := -1
		start := -1
		if startStr != "" {
			fmt.Println("start index present")
			start, _ = strconv.Atoi(startStr)
		}
		countStr := r.URL.Query().Get("count")
		if countStr != "" {
			fmt.Println("page count present")
			count, _ = strconv.Atoi(countStr)
		}

		fmt.Println("GET handler wines called")
		handlerChan := make(chan []byte)
		msg := appStateCmd{cmd: getWines, jsonReceiver: handlerChan, pageCount: count, startIndex: start}
		state.appStateChan <- msg
		resp := <-handlerChan
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

func (state *handlerState) getWineByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		fmt.Println("GET handler winebyid called")
		fmt.Println(r.URL.Path)
		pathSplit := strings.Split(r.URL.Path, "/")
		// maybe error if path is nested wrong. should have used a routing package
		id := pathSplit[len(pathSplit)-1]
		fmt.Println(id)
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
	http.HandleFunc("/status", handlerState.status)
	http.HandleFunc("/wine", handlerState.getWines)
	http.HandleFunc("/wine/", handlerState.getWineByID)

	defer ticker.Stop() // are defers necessary in main?
	defer close(handlerState.appStateChan)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
