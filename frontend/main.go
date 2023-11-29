package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type BackendState string

var (
	LeaderBackendState BackendState = "Leader"
)

type BackendResponse struct {
	State BackendState `json:"state"`
}

type follower struct {
	Name string
	URL  string
}

var backendsList = []string{"http://localhost:2221", "http://localhost:2222", "http://localhost:2223"}
var followerList = []follower{
	{
		Name: "node2",
		URL:  "localhost:1112",
	},
	{
		Name: "node3",
		URL:  "localhost:1113",
	},
}

func isLeader(addr string) bool {
	resp, err := http.Get(fmt.Sprintf("%s/raft/stats", addr))
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var backendResponse BackendResponse
	err = json.Unmarshal(body, &backendResponse)
	if err != nil {
		return false
	}
	if backendResponse.State == LeaderBackendState {
		return true
	}

	return false
}

func isAvailable(addr string) bool {
	resp, err := http.Get(fmt.Sprintf("%s/raft/stats", addr))
	if err != nil {
		return false
	}

	if resp.StatusCode == http.StatusOK {
		return true
	}

	return false
}

func main() {
	router := chi.NewRouter()

	setupCluster(":2221", followerList)

	router.Post("/api/pay", leaderProxy)
	router.Post("/api/recurring", leaderProxy)
	router.Get("/api/status/{order_id}", availableProxy)
	log.Print("frontend run")
	http.ListenAndServe(":8080", router)
}

func setupCluster(masterAddress string, followersList []follower) {
	url := fmt.Sprintf("http://localhost%s/raft/join", masterAddress)
	method := "POST"
	client := &http.Client{}

	for _, f := range followerList {

		payload := strings.NewReader(fmt.Sprintf(`{
 "node_id":"%s",
 "raft_address":"%s"
}`, f.Name, f.URL))

		req, err := http.NewRequest(method, url, payload)

		if err != nil {
			fmt.Println(err)
			return
		}
		req.Header.Add("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(body))
		time.Sleep(2 * time.Second)
	}
}

func leaderProxy(w http.ResponseWriter, r *http.Request) {
	for _, addr := range backendsList {
		if isLeader(addr) {
			url := fmt.Sprintf("%s%s", addr, r.URL.Path)
			proxyRequest(url, w, r)
			return
		}
	}
	w.WriteHeader(http.StatusBadGateway)
}

func availableProxy(w http.ResponseWriter, r *http.Request) {
	for _, addr := range backendsList {
		if isAvailable(addr) {
			url := fmt.Sprintf("%s%s", addr, r.URL.Path)
			proxyRequest(url, w, r)
			return
		}
	}
	w.WriteHeader(http.StatusBadGateway)
}

func proxyRequest(url string, w http.ResponseWriter, r *http.Request) {
	proxyReq, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	// Copy the headers from the original request to the proxy request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Send the proxy request using the custom transport
	resp, err := http.DefaultTransport.RoundTrip(proxyReq)
	if err != nil {
		http.Error(w, "Error sending proxy request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the headers from the proxy response to the original response
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set the status code of the original response to the status code of the proxy response
	w.WriteHeader(resp.StatusCode)

	// Copy the body of the proxy response to the original response
	io.Copy(w, resp.Body)
}
