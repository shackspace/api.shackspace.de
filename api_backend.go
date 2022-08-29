package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	// make sure we haven't accidently seen the alive ping yet
	last_portal_contact = time.Now().Add(-2 * portal_contact_timeout)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "www/index.htm")
	})

	http.HandleFunc("/v1/space", displaySpaceStatus)
	http.HandleFunc("/v1/online", displayShacklesStatus)
	http.HandleFunc("/v1/plena/next", displayNotImplementedYet)
	// http.HandleFunc("/v1/plena/next?redirect - get redirected directly to the newest wiki page
	http.HandleFunc("/v1/spaceapi", displaySpaceApi)
	http.HandleFunc("/v1/stats/portal", displayNotImplementedYet)

	http.HandleFunc("/v1/space/notify-open", handleNotifyOpen)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

var mutex = &sync.Mutex{}

var last_portal_state_change time.Time         // stores the time for the last state change
var last_portal_contact time.Time              // stores the time when we've last seen the space signal itself "open"
const portal_contact_timeout = 5 * time.Minute // stores the timeout after when the space is considered closed

func isShackOpen() bool {
	// lock access to shared state
	was_seen_shortly_ago := last_portal_contact.After(time.Now().Add(-portal_contact_timeout))

	return was_seen_shortly_ago
}

func notifyShackOpen() {
	mutex.Lock()
	if !isShackOpen() {
		last_portal_state_change = time.Now()
	}
	last_portal_contact = time.Now()
	mutex.Unlock()
}

func getStateChangeTime() time.Time {
	if isShackOpen() {
		return last_portal_state_change

	} else {
		return last_portal_contact.Add(portal_contact_timeout)
	}
}

func handleNotifyOpen(w http.ResponseWriter, r *http.Request) {
	api_key, err := os.ReadFile("www/auth-token.txt")
	if err != nil {
		log.Fatalln("Failed to load api auth token:", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Failed to load api auth token:")
		fmt.Fprintln(w, err)
		return
	}

	if r.URL.Query().Get("auth_token") == strings.TrimSpace(string(api_key)) {
		notifyShackOpen()
		fmt.Fprint(w, "ok")
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "invalid token")
	}
}

func displayNotImplementedYet(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	fmt.Fprintf(w, "Not implemented yet")
}

func serveJsonString(w http.ResponseWriter, value any) {

	json_string, err := json.Marshal(value)
	if err == nil {
		fmt.Fprint(w, string(json_string))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Failed to serialize json:")
		fmt.Fprintln(w, err)
	}
}

func displaySpaceStatus(w http.ResponseWriter, r *http.Request) {

	type DoorOpenState struct {
		Open bool `json:"open"`
	}
	type SlashSpaceResponse struct {
		DoorState DoorOpenState `json:"doorState"`
	}

	response := SlashSpaceResponse{
		DoorState: DoorOpenState{
			Open: isShackOpen(),
		},
	}

	serveJsonString(w, response)
}

func displayShacklesStatus(w http.ResponseWriter, r *http.Request) {
	type Api struct {
		Message string   `json:"message"`
		List    []string `json:"list"`
	}

	response := Api{
		Message: "shackles system is offline right now",
		List:    []string{},
	}

	serveJsonString(w, response)
}

func displaySpaceApi(w http.ResponseWriter, r *http.Request) {
	type SpaceApi struct {
		API   string `json:"api"`
		Space string `json:"space"`
		Logo  string `json:"logo"`
		URL   string `json:"url"`
		Icon  struct {
			Open   string `json:"open"`
			Closed string `json:"closed"`
		} `json:"icon"`
		Location struct {
			Address string  `json:"address"`
			Lon     float64 `json:"lon"`
			Lat     float64 `json:"lat"`
		} `json:"location"`
		Contact struct {
			Phone   string `json:"phone"`
			Twitter string `json:"twitter"`
			Email   string `json:"email"`
			Ml      string `json:"ml"`
			Irc     string `json:"irc"`
		} `json:"contact"`
		IssueReportChannels []string `json:"issue_report_channels"`
		State               struct {
			Icon struct {
				Open   string `json:"open"`
				Closed string `json:"closed"`
			} `json:"icon"`
			Open       bool `json:"open"`
			Lastchange int  `json:"lastchange"`
		} `json:"state"`
		Projects []string `json:"projects"`
	}

	json_string, err := os.ReadFile("www/space-api.json")
	if err != nil {
		log.Fatalln("Failed to load space api data:", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Failed to load space api data:")
		fmt.Fprintln(w, err)
		return
	}

	response := SpaceApi{}
	err = json.Unmarshal([]byte(json_string), &response)
	if err != nil {
		log.Fatalln("Failed to parse space api data:", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Failed to parse space api data:")
		fmt.Fprintln(w, err)
		return
	}

	response.State.Open = isShackOpen()
	response.State.Lastchange = int(getStateChangeTime().Unix()) // TODO: This must be better documented

	serveJsonString(w, response)
}
