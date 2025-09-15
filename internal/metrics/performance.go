package metrics

import (
    "runtime"
    "time"
)

type PerformanceMonitor struct {
    samples []PerformanceSample
    maxSamples int
}

type PerformanceSample struct {
    Timestamp   time.Time
    CPUPercent  float64
    MemoryMB    uint64
    Temperature float64 // Celsius, if available
}

func NewPerformanceMonitor() *PerformanceMonitor {
    return &PerformanceMonitor{
        maxSamples: 60, // Keep last 60 samples (1 hour at 1/min)
        samples:    make([]PerformanceSample, 0, 60),
    }
}

func (pm *PerformanceMonitor) Sample() PerformanceSample {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    sample := PerformanceSample{
        Timestamp: time.Now(),
        MemoryMB:  m.Alloc / 1024 / 1024,
        // CPU percent would need OS-specific implementation
        CPUPercent: 0,
        Temperature: 0, // Would need hardware monitoring
    }
    
    pm.addSample(sample)
    return sample
}

func (pm *PerformanceMonitor) addSample(s PerformanceSample) {
    pm.samples = append(pm.samples, s)
    if len(pm.samples) > pm.maxSamples {
        pm.samples = pm.samples[1:]
    }
}

func (pm *PerformanceMonitor) GetAverageImpact() (cpuAvg float64, memAvg uint64) {
    if len(pm.samples) == 0 {
        return 0, 0
    }
    
    var totalCPU float64
    var totalMem uint64
    
    for _, s := range pm.samples {
        totalCPU += s.CPUPercent
        totalMem += s.MemoryMB
    }
    
    return totalCPU / float64(len(pm.samples)), totalMem / uint64(len(pm.samples))
}

func (pm *PerformanceMonitor) IsSystemHealthy() bool {
    cpuAvg, memAvg := pm.GetAverageImpact()
    
    // System is healthy if average CPU < 80% and memory < 4GB
    return cpuAvg < 80.0 && memAvg < 4096
}