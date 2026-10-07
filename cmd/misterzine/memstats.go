package main

import (
	"runtime"
	"time"
)

// memStats is the debug API's view of the Go heap: what the collector has
// to do while the screen moves. Reading it stops the world for a moment,
// so it is read only when asked.
func memStats() map[string]any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]any{
		"heap_alloc_mb":  float64(m.HeapAlloc) / (1 << 20),
		"heap_sys_mb":    float64(m.HeapSys) / (1 << 20),
		"next_gc_mb":     float64(m.NextGC) / (1 << 20),
		"num_gc":         m.NumGC,
		"gc_pause_ms":    float64(m.PauseTotalNs) / float64(time.Millisecond),
		"gc_cpu_percent": m.GCCPUFraction * 100,
		"goroutines":     runtime.NumGoroutine(),
	}
}
