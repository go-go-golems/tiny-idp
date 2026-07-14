package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
)

type report struct {
	Workers           int                   `json:"workers"`
	Attempts          int                   `json:"attempts"`
	Elapsed           time.Duration         `json:"elapsed"`
	AttemptsPerSecond float64               `json:"attempts_per_second"`
	MemoryBefore      uint64                `json:"memory_before_bytes"`
	MemoryAfter       uint64                `json:"memory_after_bytes"`
	MemoryPeak        uint64                `json:"memory_peak_bytes"`
	Stats             idp.PasswordWorkStats `json:"password_work"`
}

func main() {
	workers := flag.Int("workers", 8, "concurrent authentication workers")
	attempts := flag.Int("attempts", 24, "total password attempts")
	maxConcurrent := flag.Int("max-concurrent", 2, "maximum concurrent Argon2id operations")
	logLevel := flag.String("log-level", "info", "zerolog level")
	flag.Parse()
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --log-level: %v\n", err)
		os.Exit(2)
	}
	log.Logger = zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
	if *workers < 1 || *attempts < 1 || *maxConcurrent < 1 {
		log.Fatal().Msg("workers, attempts, and max-concurrent must be positive")
	}
	if err := run(context.Background(), *workers, *attempts, *maxConcurrent); err != nil {
		log.Fatal().Err(err).Msg("password work load failed")
	}
}

func run(ctx context.Context, workers, attempts, maxConcurrent int) error {
	store := memory.New()
	policy := idpaccounts.DefaultLoginPolicy()
	policy.LockoutThreshold = 1_000_000
	service, err := idpaccounts.NewService(store, idpaccounts.Options{
		LoginPolicy:  policy,
		PasswordWork: idp.PasswordWorkConfig{MaxConcurrent: maxConcurrent},
		Audit:        idp.NewMemorySink(),
	})
	if err != nil {
		return err
	}
	if _, err := service.Create(ctx, idpaccounts.CreateRequest{ID: "load-user", Subject: "load-subject", Login: "load-user", Password: []byte("a production load password phrase")}); err != nil {
		return err
	}
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	jobs := make(chan struct{})
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup
	start := time.Now()
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				_, err := service.AuthenticatePassword(ctx, "load-user", "an incorrect password phrase", idp.LoginMetadata{ClientID: "load-client", RemoteAddr: "192.0.2.1"})
				if !errors.Is(err, idpaccounts.ErrInvalidCredentials) {
					errorsCh <- err
					return
				}
			}
		}()
	}
	for attempt := 0; attempt < attempts; attempt++ {
		jobs <- struct{}{}
	}
	close(jobs)
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			return err
		}
	}
	elapsed := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	result := report{
		Workers:           workers,
		Attempts:          attempts,
		Elapsed:           elapsed,
		AttemptsPerSecond: float64(attempts) / elapsed.Seconds(),
		MemoryBefore:      before.Alloc,
		MemoryAfter:       after.Alloc,
		MemoryPeak:        after.Sys,
		Stats:             service.PasswordWorkStats(),
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
