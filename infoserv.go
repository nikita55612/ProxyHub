package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var infoTsLast atomic.Int64
var infoCache = ""

var statTsLast atomic.Int64
var statCache = ""

type InfoServerParams struct {
	Host string
	Port int
}

type VnStatData struct {
	VnStatVersion string            `json:"vnstatversion"`
	JsonVersion   string            `json:"jsonversion"`
	Interfaces    []VnStatInterface `json:"interfaces"`
}

type VnStatInterface struct {
	Name    string         `json:"name"`
	Alias   string         `json:"alias"`
	Created VnStatTimeData `json:"created"`
	Updated VnStatTimeData `json:"updated"`
	Traffic VnStatTraffic  `json:"traffic"`
}

type VnStatTimeData struct {
	Date      VnStatDate  `json:"date"`
	Time      *VnStatTime `json:"time,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

type VnStatDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type VnStatTime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type VnStatTraffic struct {
	Total TrafficStats `json:"total"`
	Day   []DayStats   `json:"day"`
}

type TrafficStats struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
}

type DayStats struct {
	ID        int        `json:"id"`
	Date      VnStatDate `json:"date"`
	Timestamp int64      `json:"timestamp"`
	Rx        uint64     `json:"rx"`
	Tx        uint64     `json:"tx"`
}

type Stat struct {
	DayRx   uint64 `json:"dayRx"`
	DayTX   uint64 `json:"dayTx"`
	Day7Rx  uint64 `json:"day7Rx"`
	Day7TX  uint64 `json:"day7Tx"`
	Day30Rx uint64 `json:"day30Rx"`
	Day30TX uint64 `json:"day30Tx"`
}

func execCommand(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	stdout, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(stdout)
}

func infoHandle(w http.ResponseWriter, r *http.Request) {
	tsNow := time.Now().Unix()
	if tsNow-infoTsLast.Load() <= 4 {
		fmt.Fprint(w, infoCache)
		return
	}

	header := w.Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	header.Set("Access-Control-Allow-Headers", "Content-Type")

	infoCache = ""
	infoCache += strings.ReplaceAll(strings.ReplaceAll(execCommand("fastfetch", "--pipe", "--structure", "separator:os:separator:host:kernel:uptime:packages:shell:de:wm:wmtheme:theme:icons:font:cpu:gpu:memory:disk:localip"), "[34C", ""), "[31C", "")
	infoCache += execCommand("vnstat")
	infoCache += execCommand("vnstat", "-h")
	infoCache += execCommand("vnstat", "-hg")
	infoCache += execCommand("vnstat", "-5")

	infoTsLast.Store(tsNow)

	fmt.Fprint(w, infoCache)
}

func statHandle(w http.ResponseWriter, r *http.Request) {
	tsNow := time.Now().Unix()
	if tsNow-statTsLast.Load() <= 12 {
		fmt.Fprint(w, statCache)
		return
	}

	header := w.Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	header.Set("Access-Control-Allow-Headers", "Content-Type")

	exc := execCommand("vnstat", "--json", "d", "30")
	if exc == "" {
		http.Error(w, "exec command error", http.StatusBadRequest)
		return
	}

	var vnStat VnStatData
	err := json.Unmarshal([]byte(exc), &vnStat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := new(Stat)

	if len(vnStat.Interfaces) == 0 {
		http.Error(w, "empty data", http.StatusBadRequest)
		return
	}

	days := vnStat.Interfaces[0].Traffic.Day

	if len(days) > 0 {
		lastDay := days[len(days)-1]
		result.DayRx = lastDay.Rx
		result.DayTX = lastDay.Tx
	}

	for i, day := range days {
		result.Day30Rx += day.Rx
		result.Day30TX += day.Tx
		if i >= len(days)-7 {
			result.Day7Rx += day.Rx
			result.Day7TX += day.Tx
		}
	}

	header.Set("Content-Type", "application/json")

	b, _ := json.Marshal(&result)

	statCache = string(b)
	statTsLast.Store(tsNow)

	fmt.Fprint(w, statCache)
}

func rawStatHandle(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	header.Set("Access-Control-Allow-Headers", "Content-Type")

	query := r.URL.Query()
	mode := query.Get("mode")
	if mode == "" {
		mode = "d"
	}
	limit := query.Get("limit")
	if limit == "" {
		limit = "30"
	}
	li, err := strconv.Atoi(limit)
	if err == nil {
		if li > 90 {
			http.Error(w, "error", http.StatusBadRequest)
			return
		}
	}
	result := execCommand("vnstat", "--json", mode, limit)
	if result == "" {
		http.Error(w, "exec command error", http.StatusBadRequest)
		return
	}

	header.Set("Content-Type", "application/json")

	fmt.Fprint(w, result)
}

func RunInfoServer(ctx context.Context, stop context.CancelFunc, params *InfoServerParams) {
	defer stop()

	mux := http.NewServeMux()

	mux.HandleFunc("/info", infoHandle)
	mux.HandleFunc("/stat", statHandle)
	mux.HandleFunc("/rawstat", rawStatHandle)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pong")
	})

	addr := fmt.Sprintf("%s:%d", params.Host, params.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("Info server running [LOCAL] at http://127.0.0.1:%d\n", params.Port)
	log.Printf("Info server running [GLOBAL] at http://%s:%d\n", ipAddr, params.Port)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down HTTP info server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Info server shutdown error: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Info server failed: %v", err)
	}

	log.Println("HTTP info server stopped gracefully.")
}
