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

func timeFunc(ticker *time.Ticker, appState *appState) {
	//prevSuccess := 0
	//prevErrors := 0
	prevRequests := 0
	//prevAvailability := 0
	for {
		select {
		case <-ticker.C:
			requests := appState.requests
			minuteRequests := requests - prevRequests
			prevRequests = requests
			fmt.Printf("requests this minute: %d\n", minuteRequests)
			fmt.Printf("number of wines: %d\n", len(appState.csvRecords))
			fmt.Println("Timer Called")
			fmt.Fprintf(os.Stderr, "err test")
		}
	}
}

// trying to manage shared state w/ channels
// handlers send over shared channel
// and provide a channel with which they will receive

// named indeces for csv rows
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

//cmd types handlers might send over channel
type cmdType int

const (
	incrementRequests  cmdType = 0
	incrementErrors    cmdType = 1
	incrementSuccesses cmdType = 2
	putWine            cmdType = 3
	getStatus          cmdType = 4
	getWines           cmdType = 5
	getWineByID        cmdType = 6
)

// app-wide state that needs synchronized access
type appState struct {
	requests     int
	successes    int
	errors       int
	availability int
	numWines     int
	csvRecords   [][]string
	status       bool
}

// cmd to send to the appState handler
type appStateCmd struct {
	cmd          cmdType
	wineID       int
	putRecord    putWineData
	startIndex   int
	pageCount    int
	jsonReceiver chan *appStateResponse
}

// give handlers access to the appState manager's channel
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

type appStateResponse struct {
	success bool
	message string
	payload []byte
}

type putWineData struct {
	Country             string `json:"country"`
	Description         string `json:"description"`
	Designation         string `json:"designation"`
	Points              string `json:"points"`
	Price               string `json:"price"`
	Province            string `json:"province"`
	Region1             string `json:"region_1"`
	Region2             string `json:"region_2"`
	TasterName          string `json:"taster_name"`
	TasterTwitterHandle string `json:"taster_twitter_handle"`
	Title               string `json:"title"`
	Variety             string `json:"variety"`
	Winery              string `json:"winery"`
}

// counter manager will:
// - start a goroutine that listens on a channel to requests to touch app state
// - return that channel
func startCounterManager(appState *appState) chan<- appStateCmd {

	appStateChan := make(chan appStateCmd)

	// start anonymous goroutine
	go func() {
		// has access to appStateChan via closure
		for cmd := range appStateChan {
			switch cmd.cmd {
			case incrementRequests:
				appState.requests++
			case incrementSuccesses:
				appState.successes++
			case incrementErrors:
				appState.errors++
			case getStatus:
				t := time.Now()
				ts := t.Format(time.RFC3339)
				if appState.status {
					resp := &statusResponse{Status: "ok", Msg: "", Ts: ts}
					respBytes, err := json.Marshal(resp)
					if err != nil {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "json marshalling error"}
					}
					cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: respBytes}

				} else {
					resp := &statusResponse{Status: "error", Msg: "failed to load csv", Ts: ts}
					respBytes, err := json.Marshal(resp)
					if err != nil {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "json marshalling error"}
					}
					cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: respBytes}
				}
			case getWines:
				if (cmd.startIndex < 0) || (cmd.pageCount < 0) {
					wineList := make([]wineResponse, len(appState.csvRecords[1:]))
					for i, record := range appState.csvRecords[1:] {
						newRecord := wineResponse{ID: record[id], Title: record[title]}
						wineList[i] = newRecord
					}
					resp := wineListResponse{Wines: wineList}
					respBytes, _ := json.Marshal(resp)
					cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: respBytes}
				} else {
					if cmd.startIndex > len(appState.csvRecords[1:]) {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "index out of range"}
					} else if (cmd.startIndex + cmd.pageCount) > len(appState.csvRecords[1:]) {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "index + count out of range"}
					} else {
						wineListSlice := appState.csvRecords[cmd.startIndex+1 : cmd.startIndex+cmd.pageCount+1]
						wineList := make([]wineResponse, len(wineListSlice))
						for i, record := range wineListSlice {
							newRecord := wineResponse{ID: record[id], Title: record[title]}
							wineList[i] = newRecord
						}
						resp := wineListResponse{Wines: wineList}
						respBytes, err := json.Marshal(resp)
						if err != nil {
							cmd.jsonReceiver <- &appStateResponse{success: false, message: "json marshalling error"}
						}
						cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: respBytes}
					}

				}
			case getWineByID:
				if (cmd.wineID < 0) || (cmd.wineID >= (len(appState.csvRecords) - 1)) {
					cmd.jsonReceiver <- &appStateResponse{success: false, message: "id not available"}
				} else {
					wine := appState.csvRecords[cmd.wineID+1]
					resp := wineResponse{ID: wine[id], Title: wine[title]}
					respBytes, err := json.Marshal(resp)
					if err != nil {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "json marshalling error"}
					}
					cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: respBytes}
				}

			case putWine:
				wineToPut := cmd.putRecord
				lenString := strconv.Itoa(len(appState.csvRecords) - 1)
				fmt.Println(lenString)
				newRecord := make([]string, 14)
				newRecord[id] = lenString
				newRecord[country] = wineToPut.Country
				newRecord[description] = wineToPut.Description
				newRecord[designation] = wineToPut.Designation
				newRecord[points] = wineToPut.Points
				newRecord[price] = wineToPut.Price
				newRecord[province] = wineToPut.Province
				newRecord[region1] = wineToPut.Region1
				newRecord[region2] = wineToPut.Region2
				newRecord[tasterName] = wineToPut.TasterName
				newRecord[tasterTwitterHandle] = wineToPut.TasterTwitterHandle
				newRecord[title] = wineToPut.Title
				newRecord[variety] = wineToPut.Variety
				newRecord[winery] = wineToPut.Winery
				appState.csvRecords = append(appState.csvRecords, newRecord)
				cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: []byte(`{"status":"successful put"}`)}
			}
		}
	}()
	return appStateChan
}

func (state *handlerState) status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: getStatus, jsonReceiver: handlerChan}
		state.appStateChan <- msg

		resp := <-handlerChan
		if resp.success {
			w.WriteHeader(http.StatusOK)
			w.Write(resp.payload)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"message": "not allowed"}`))
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
		//handlerChan := make(chan []byte)
		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: getWines, jsonReceiver: handlerChan, pageCount: count, startIndex: start}
		state.appStateChan <- msg

		resp := <-handlerChan
		if resp.success {
			w.WriteHeader(http.StatusOK)
			w.Write(resp.payload)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}
	case "PUT":
		decoder := json.NewDecoder(r.Body)
		var myData putWineData
		err := decoder.Decode(&myData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"couldn't decode json"}`))
		}
		fmt.Println(myData)
		fmt.Println(myData.Winery)

		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: putWine, jsonReceiver: handlerChan, putRecord: myData}
		state.appStateChan <- msg

		resp := <-handlerChan
		if resp.success {
			w.WriteHeader(http.StatusAccepted)
			w.Write(resp.payload)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"message": "not allowed"}`))
	}
}

func (state *handlerState) getWineByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		pathSplit := strings.Split(r.URL.Path, "/")
		id, err := strconv.Atoi(pathSplit[len(pathSplit)-1])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"couldn't prase out id from path"}`))
		}
		fmt.Println(id)
		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: getWineByID, jsonReceiver: handlerChan, wineID: id}
		state.appStateChan <- msg
		resp := <-handlerChan
		if resp.success {
			w.WriteHeader(http.StatusOK)
			w.Write(resp.payload)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"message": "not allowed"}`))
	}
}

func main() {
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

	appState := &appState{csvRecords: records, status: !csvError}
	fmt.Println("records:", len(records))

	ticker := time.NewTicker(time.Second * 5)

	//timer func doesn't need thread-safe access to appState
	go timeFunc(ticker, appState)

	// this is so wild, i need to know what's happening here
	handlerState := handlerState{
		appStateChan: startCounterManager(appState)}

	// investigate better approaches to giving these funcs access to appStateChan
	http.HandleFunc("/status", handlerState.status)
	http.HandleFunc("/wine", handlerState.getWines)
	http.HandleFunc("/wine/", handlerState.getWineByID)

	defer ticker.Stop() // are defers necessary in main?
	defer close(handlerState.appStateChan)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
