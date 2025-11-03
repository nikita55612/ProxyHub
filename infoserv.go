package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var tsLast atomic.Int64
var infoCache = ""

type InfoServerParams struct {
	Host string
	Port int
}

type VnStatData struct {
	VnStatVersion string            `json:"vnstatversion"`
	JsonVersion   int               `json:"jsonversion"`
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
	if tsNow-tsLast.Load() <= 3 {
		fmt.Fprint(w, infoCache)
		return
	}
	tsLast.Store(tsNow)
	infoCache = ""
	infoCache += strings.ReplaceAll(strings.ReplaceAll(execCommand("fastfetch", "--pipe", "--structure", "separator:os:separator:host:kernel:uptime:packages:shell:de:wm:wmtheme:theme:icons:font:cpu:gpu:memory:disk:localip"), "[34C", ""), "[31C", "")
	infoCache += execCommand("vnstat")
	infoCache += execCommand("vnstat", "-h")
	infoCache += execCommand("vnstat", "-hg")
	infoCache += execCommand("vnstat", "-5")

	fmt.Fprint(w, infoCache)
}

func statHandle(w http.ResponseWriter, r *http.Request) {
	exc := execCommand("vnstat --json d 30")
	if exc == "" {
		http.Error(w, "error", http.StatusBadRequest)
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

	slices.Reverse(vnStat.Interfaces[0].Traffic.Day)

	if len(vnStat.Interfaces[0].Traffic.Day) > 0 {
		result.DayRx = vnStat.Interfaces[0].Traffic.Day[0].Rx
		result.DayTX = vnStat.Interfaces[0].Traffic.Day[0].Tx
	}

	for i, day := range vnStat.Interfaces[0].Traffic.Day {
		result.Day30Rx += day.Rx
		result.Day30TX += day.Tx
		if i < 7 {
			result.Day7Rx += day.Rx
			result.Day7TX += day.Tx
		}
	}

	header := w.Header()
	header.Set("Content-Type", "application/json")

	b, _ := json.Marshal(&result)

	fmt.Fprint(w, string(b))
}

func rawStatHandle(w http.ResponseWriter, r *http.Request) {
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
	commandPrompt := fmt.Sprintf("vnstat --json %s %s", mode, limit)
	result := execCommand(commandPrompt)
	if result == "" {
		http.Error(w, "error", http.StatusBadRequest)
		return
	}

	header := w.Header()
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

	log.Printf("Info server running [LOCAL] at http://%s\n", addr)
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
