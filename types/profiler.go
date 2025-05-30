package types

import "time"

type Profiler struct {
	StartTime   time.Time
	EndProfiler func(Profiler) time.Duration
}

func __profilerEnd(profile Profiler) time.Duration {

	return time.Since(profile.StartTime)

}

func ProfilerStart(id string) Profiler {

	return Profiler{
		StartTime:   time.Now(),
		EndProfiler: __profilerEnd,
	}

}
