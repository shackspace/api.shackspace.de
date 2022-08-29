package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed www/index.htm
var index_htm_bytes string

// file path to the space api template. must follow the space api format and will be provided
// under `/v1/spaceapi`. The current open status will be updated before serving, the file is
// not touched though.
var space_api_path string

// file path to a file that contains the api token for `/v1/space/notify-open`.
// api requests will only be accepted for this path when the `auth_token` query is
// this files content.
// NOTE: Leading and trailing whitespace will be removed from the file contents.
var auth_token_path string

// file path to a file that contains the time stamp of the last sent status update.
// NOTE: This file will be updated with each successful request.
var status_db_path string

func main() {
	err := parseCli()
	if err != nil {
		log.Fatal("could not parse command line: ", err)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, index_htm_bytes)
	})

	http.HandleFunc("/v1/space", displaySpaceStatus)
	http.HandleFunc("/v1/online", displayShacklesStatus)
	http.HandleFunc("/v1/plena/next", displayNextPlenum)
	// http.HandleFunc("/v1/plena/next?redirect - get redirected directly to the newest wiki page
	http.HandleFunc("/v1/spaceapi", displaySpaceApi)
	http.HandleFunc("/v1/stats/portal", displayNotImplementedYet)

	http.HandleFunc("/v1/space/notify-open", handleNotifyOpen)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

// parses the command line arguments and initializes the global variables.
// will return an error if anything went wrong.
func parseCli() error {
	argv := os.Args

	if len(argv) < 4 {
		return errors.New("requires arguments\nusage: api <space_api_def> <api_token> <status db>")
	}

	space_api_path = argv[1]
	auth_token_path = argv[2]
	status_db_path = argv[3]

	_, err := os.ReadFile(space_api_path)
	if err != nil {
		log.Fatal("could not open space api file")
		return err
	}

	_, err = os.ReadFile(auth_token_path)
	if err != nil {
		log.Fatal("could not open auth token path")
		return err
	}

	// initialize last time stamp from status path
	init_timestamp, err := os.ReadFile(status_db_path)
	if errors.Is(err, os.ErrNotExist) {
		_ = defaultInitalizeStatusDb()
	} else if err == nil {

		intval, err := strconv.ParseInt(string(init_timestamp), 10, 64)
		if err == nil {
			// make sure we haven't accidently seen the alive ping yet.
			last_portal_contact = time.Unix(intval, 0)
		} else {
			log.Fatal("failed to read status db, please make sure it's readable and contains a valid unix timestamp!")
			return err
		}

	} else {
		log.Fatal("could not open status db path")
		return err
	}

	return nil
}

func defaultInitalizeStatusDb() error {
	// make sure we haven't accidently seen the alive ping yet.
	last_portal_contact = time.Now().Add(-2 * portal_contact_timeout)

	return writeStatusDb()
}

func writeStatusDb() error {
	err := os.WriteFile(status_db_path, []byte(strconv.FormatInt(last_portal_contact.Unix(), 10)), 0o666)
	if err != nil {
		log.Fatalln("Failed to write status db: ", err)
		return err
	}
	return nil
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

	_ = writeStatusDb()

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
	api_key, err := os.ReadFile(auth_token_path)
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

	json_string, err := os.ReadFile(space_api_path)
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

// Computes the date of the Plenum for the week `timestamp` is in.
// Returns the start of that day.
func computePlenumForWeek(timestamp time.Time) time.Time {
	day := time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, timestamp.Location())

	start_of_week := day.Add(time.Duration(-24 * int64(time.Hour) * int64(day.Weekday())))

	_, week := timestamp.ISOWeek()

	var weekday time.Weekday
	if week%2 == 0 {
		weekday = time.Thursday
	} else {
		weekday = time.Wednesday
	}

	plenum_date := start_of_week.Add(time.Duration(24 * int64(time.Hour) * int64(weekday)))

	return plenum_date
}

func displayNextPlenum(w http.ResponseWriter, r *http.Request) {
	type PlenumInfo struct {
		Date    time.Time `json:"date"`
		FromNow string    `json:"fromNow"`
		URL     string    `json:"url"`
	}

	now := time.Now().Local()

	plenum_date := computePlenumForWeek(now)

	// If we already missed the plenum this week,
	// we have to provide the date for next week.
	if plenum_date.Before(now) {
		plenum_date.Add(7 * 24 * time.Hour)
		plenum_date = computePlenumForWeek(plenum_date)
	}

	// adjust this to configure the plenum time!
	plenum_date.Add(19 * time.Hour)

	response := PlenumInfo{
		Date:    plenum_date,
		FromNow: "soooooon",
		URL:     fmt.Sprintf("https://wiki.shackspace.de/plenum/%04d-%02d-%02d", plenum_date.Year(), plenum_date.Month(), plenum_date.Day()),
	}

	if r.URL.Query().Has("redirect") {
		w.Header().Add("Location", response.URL)
		w.Header().Add("Content-Type", "text/html; charset=utf-8")

		w.WriteHeader(http.StatusFound)

		fmt.Fprintf(w, "Redirecting to <a href=\"%s\">%s</a>.", response.URL, response.URL)

		return
	}

	serveJsonString(w, response)
}
