package server

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type systemStatus struct {
	Status     string       `json:"status"`
	Version    string       `json:"version"`
	Uptime     string       `json:"uptime"`
	GoVersion  string       `json:"goVersion"`
	NumCPU     int          `json:"numCpu"`
	Goroutines int          `json:"goroutines"`
	Memory     memoryStatus `json:"memory"`
}

type memoryStatus struct {
	Alloc      uint64 `json:"allocBytes"`
	TotalAlloc uint64 `json:"totalAllocBytes"`
	Sys        uint64 `json:"sysBytes"`
	NumGC      uint32 `json:"numGc"`
}

func (s *Server) handleSystemStatus(w http.ResponseWriter, _ *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := systemStatus{
		Status:     "healthy",
		Version:    "dev",
		Uptime:     time.Since(startTime).Round(time.Second).String(),
		GoVersion:  runtime.Version(),
		NumCPU:     runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
		Memory: memoryStatus{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
