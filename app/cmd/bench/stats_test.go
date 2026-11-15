package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPercentileNearestRank(t *testing.T) {
	data := make([]time.Duration, 30)
	for i := range data {
		data[i] = time.Duration(i + 1)
	}
	if got := percentile(data, 0.95); got != 29 {
		t.Fatalf("percentile = %d, want 29", got)
	}
}

func TestMedian(t *testing.T) {
	data := make([]time.Duration, 30)
	for i := range data {
		data[i] = time.Duration((i + 1) * 2)
	}
	if got := median(data); got != 31 {
		t.Fatalf("median even = %d, want 31 (15.5 in doubled units)", got)
	}
	if got := median([]time.Duration{1, 2, 3}); got != 2 {
		t.Fatalf("median odd = %d, want 2", got)
	}
	if got := median([]time.Duration{7}); got != 7 {
		t.Fatalf("median single = %d, want 7", got)
	}
	if got := median([]time.Duration{15, 16}); got != 15 {
		t.Fatalf("median even duration truncation = %d, want 15", got)
	}
}

func TestMeasure(t *testing.T) {
	calls := 0
	samples, err := measure(3, 2, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 5 || len(samples) != 3 {
		t.Fatalf("calls=%d samples=%d, want 5 and 3", calls, len(samples))
	}
}

func TestMeasureStopsOnError(t *testing.T) {
	want := errors.New("stop")
	_, err := measure(3, 1, func() error {
		return want
	})
	if err != want {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestMeasurePreservesExecutionOrder(t *testing.T) {
	calls := 0
	samples, err := measure(3, 0, func() error {
		calls++
		time.Sleep(time.Duration(calls) * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !(samples[0] < samples[1] && samples[1] < samples[2]) {
		t.Fatalf("samples are not in execution order: %v", samples)
	}
}

func TestMeasureInterleavedOrder(t *testing.T) {
	var order []string
	cases := []benchCase{
		{metric: "a", fn: func() error { order = append(order, "A"); return nil }},
		{metric: "b", fn: func() error { order = append(order, "B"); return nil }},
	}
	results, err := measureInterleaved(cases, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(results[0]), 3; got != want {
		t.Fatalf("case A samples = %d, want %d", got, want)
	}
	if got, want := len(results[1]), 3; got != want {
		t.Fatalf("case B samples = %d, want %d", got, want)
	}
	if got, want := strings.Join(order, ""), "ABABABAB"; got != want {
		t.Fatalf("execution order = %q, want %q", got, want)
	}
}
