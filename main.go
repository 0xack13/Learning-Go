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

// named indices for csv rows
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

//cmd types handled by the appState goroutine
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

// app-wide state that http handler threads need synchronized access to
type appState struct {
	requests     int
	successes    int
	errors       int
	availability int
	numWines     int
	csvRecords   [][]string
	status       bool
}

// cmd sent to the appState manager
type appStateCmd struct {
	cmd          cmdType
	wineID       int
	putRecord    putWineData
	start        string
	count        string
	jsonReceiver chan *appStateResponse
}

//response msg from app state manager. will also indicate success of the operation
type appStateResponse struct {
	success bool
	message string
	payload []byte
}

// used as receiver on http handler methods to give access to the appState manager's channel
// making the channel global felt weird
type handlerState struct {
	appStateChan chan<- *appStateCmd
}

//have to use field tags to get lowercase
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

func timeFunc(ticker *time.Ticker, appState *appState) {
	/*
		Called every minute
		Print metrics to stderr
		Only reads from appState, so no synchronized access required
	*/
	prevSuccesses := 0
	prevErrors := 0
	prevRequests := 0
	for {
		select {
		case <-ticker.C:
			requests := appState.requests
			minuteRequests := requests - prevRequests
			prevRequests = requests
			fmt.Fprintf(os.Stderr, "requests this minute: %d\n", minuteRequests)

			errors := appState.errors
			minuteErrors := errors - prevErrors
			prevErrors = errors
			fmt.Fprintf(os.Stderr, "errors this minute: %d\n", minuteErrors)

			successes := appState.successes
			minuteSuccesses := successes - prevSuccesses
			prevSuccesses = successes
			fmt.Fprintf(os.Stderr, "succuessful requests this minute: %d\n", minuteSuccesses)

			if minuteRequests > 0 {
				fmt.Fprintf(os.Stderr, "availability for this minute: %f\n", (float64(minuteSuccesses) / float64(minuteRequests) * 100))
			} else {
				fmt.Fprintf(os.Stderr, "availability for this minute: No requests yet this minute\n")
			}

			fmt.Fprintf(os.Stderr, "number of wines: %d\n", (len(appState.csvRecords) - 1))
		}
	}
}

func startAppStateManager(appState *appState) chan<- *appStateCmd {
	/*
		   app state manager will:
			 - start a goroutine that listens to a single channel & synchronizes to all requests to touch app state
			   - we're avoiding race conditions
		     - returns that channel to main
	*/

	appStateChan := make(chan *appStateCmd)

	// start anonymous goroutine
	go func() {
		// goroutine has access to appStateChan via closure
		for cmd := range appStateChan {
			switch cmd.cmd {
			case incrementRequests:
				appState.requests++
			case incrementSuccesses:
				appState.successes++
			case incrementErrors:
				appState.errors++
			case getStatus:
				fmt.Printf("Checking status of wine csv load\n")
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
				if (cmd.start == "") || (cmd.count == "") {
					fmt.Printf("getting all wines\n")
					wineList := make([]wineResponse, len(appState.csvRecords[1:]))
					for i, record := range appState.csvRecords[1:] {
						newRecord := wineResponse{ID: record[id], Title: record[title]}
						wineList[i] = newRecord
					}
					resp := wineListResponse{Wines: wineList}
					respBytes, _ := json.Marshal(resp)
					cmd.jsonReceiver <- &appStateResponse{success: true, message: "", payload: respBytes}
				} else {
					startNum, err := strconv.Atoi(cmd.start)
					if err != nil {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "invalid start param"}
						break
					}
					countNum, err := strconv.Atoi(cmd.count)
					if err != nil {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "invalid count param"}
						break
					}
					fmt.Printf("getting wines with start %s and count %s\n", cmd.start, cmd.count)
					if (startNum > len(appState.csvRecords[1:])) || (startNum < 0) {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "index out of range"}
						break
					}
					if (startNum+countNum) > len(appState.csvRecords[1:]) || ((startNum + countNum) < startNum) {
						cmd.jsonReceiver <- &appStateResponse{success: false, message: "index + count out of range"}
						break
					}

					wineListSlice := appState.csvRecords[startNum+1 : startNum+countNum+1]
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
			case getWineByID:
				fmt.Printf("Attempting to retrieve wine with id %d\n", cmd.wineID)
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
				fmt.Printf("Adding a new wine: %s\n", wineToPut.Title)
				lenString := strconv.Itoa(len(appState.csvRecords) - 1)
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
	state.appStateChan <- &appStateCmd{cmd: incrementRequests}
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: getStatus, jsonReceiver: handlerChan}
		state.appStateChan <- &msg

		resp := <-handlerChan
		if resp.success {
			state.appStateChan <- &appStateCmd{cmd: incrementSuccesses}
			w.WriteHeader(http.StatusOK)
			w.Write(resp.payload)
		} else {
			state.appStateChan <- &appStateCmd{cmd: incrementErrors}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}
	default:
		state.appStateChan <- &appStateCmd{cmd: incrementErrors}
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"message": "not allowed"}`))
	}
}

func (state *handlerState) getOrPutWines(w http.ResponseWriter, r *http.Request) {
	state.appStateChan <- &appStateCmd{cmd: incrementRequests}
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		start := r.URL.Query().Get("start")
		count := r.URL.Query().Get("count")

		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: getWines, jsonReceiver: handlerChan, count: count, start: start}
		state.appStateChan <- &msg

		resp := <-handlerChan
		if resp.success {
			state.appStateChan <- &appStateCmd{cmd: incrementSuccesses}
			w.WriteHeader(http.StatusOK)
			w.Write(resp.payload)
		} else {
			state.appStateChan <- &appStateCmd{cmd: incrementErrors}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}
	case "PUT":
		decoder := json.NewDecoder(r.Body)
		var myData putWineData
		err := decoder.Decode(&myData)
		if err != nil {
			state.appStateChan <- &appStateCmd{cmd: incrementErrors}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"couldn't decode json"}`))
			return
		}

		if myData.Title == "" {
			state.appStateChan <- &appStateCmd{cmd: incrementErrors}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"no valid title in wine json"}`))
			return

		}

		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: putWine, jsonReceiver: handlerChan, putRecord: myData}
		state.appStateChan <- &msg

		resp := <-handlerChan
		if resp.success {
			state.appStateChan <- &appStateCmd{cmd: incrementSuccesses}
			w.WriteHeader(http.StatusAccepted)
			w.Write(resp.payload)
		} else {
			state.appStateChan <- &appStateCmd{cmd: incrementErrors}
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(resp.message))
		}

	default:
		state.appStateChan <- &appStateCmd{cmd: incrementErrors}
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"message": "not allowed"}`))
	}
}

func (state *handlerState) getWineByID(w http.ResponseWriter, r *http.Request) {
	state.appStateChan <- &appStateCmd{cmd: incrementRequests}
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		pathSplit := strings.Split(r.URL.Path, "/")
		id, err := strconv.Atoi(pathSplit[len(pathSplit)-1])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"couldn't prase out id from path"}`))
		}
		handlerChan := make(chan *appStateResponse)
		msg := appStateCmd{cmd: getWineByID, jsonReceiver: handlerChan, wineID: id}
		state.appStateChan <- &msg
		resp := <-handlerChan
		if resp.success {
			state.appStateChan <- &appStateCmd{cmd: incrementSuccesses}
			w.WriteHeader(http.StatusOK)
			w.Write(resp.payload)
		} else {
			state.appStateChan <- &appStateCmd{cmd: incrementErrors}
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
		fmt.Printf("Couldn't open the csv file: %s\n", err)
		csvError = true
	}
	r := csv.NewReader(csvfile)
	records, err := r.ReadAll()
	if err != nil {
		fmt.Printf("Couldn't parse the csv records: %s\n", err)
		csvError = true
	}

	appState := &appState{csvRecords: records, status: !csvError}
	fmt.Println("Initial count of records: ", (len(records) - 1))

	ticker := time.NewTicker(time.Second * 60)

	// start metrics ticker in a goroutine
	go timeFunc(ticker, appState)

	//call handlefuncs as methods on handler state. this is still weird to me.
	handlerState := handlerState{
		appStateChan: startAppStateManager(appState)}

	http.HandleFunc("/status", handlerState.status)
	http.HandleFunc("/wine", handlerState.getOrPutWines)
	http.HandleFunc("/wine/", handlerState.getWineByID)

	// still unsure if these defers are necessary from main
	defer ticker.Stop()
	defer close(handlerState.appStateChan)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
