package main

import (
	"fmt"
	"math"
	"time"
)

type benchCase struct {
	metric, profile, artifact string
	fn                        func() error
}

func measureInterleaved(cases []benchCase, n, warmup int) ([][]time.Duration, error) {
	if len(cases) == 0 || n <= 0 || warmup < 0 {
		return nil, fmt.Errorf("invalid interleaved measurement counts")
	}
	for i := 0; i < warmup; i++ {
		for _, c := range cases {
			if err := c.fn(); err != nil {
				return nil, err
			}
		}
	}
	results := make([][]time.Duration, len(cases))
	for i := range results {
		results[i] = make([]time.Duration, 0, n)
	}
	for i := 0; i < n; i++ {
		for j, c := range cases {
			start := time.Now()
			if err := c.fn(); err != nil {
				return nil, err
			}
			results[j] = append(results[j], time.Since(start))
		}
	}
	return results, nil
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 || p <= 0 || p > 1 {
		return 0
	}
	index := int(math.Ceil(p*float64(len(sorted)))) - 1
	return sorted[index]
}

func median(sorted []time.Duration) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

func measure(n, warmup int, fn func() error) ([]time.Duration, error) {
	if n <= 0 || warmup < 0 {
		return nil, fmt.Errorf("invalid measurement counts: n=%d warmup=%d", n, warmup)
	}
	for i := 0; i < warmup; i++ {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		if err := fn(); err != nil {
			return nil, err
		}
		samples = append(samples, time.Since(start))
	}
	return samples, nil
}
