package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joluc/weather-exporter/internal/collector"
	"github.com/joluc/weather-exporter/internal/config"
	"github.com/joluc/weather-exporter/internal/provider"
)

func main() {
	var cities config.CityFlags
	listenAddress := flag.String("listen-address", ":9798", "HTTP listen address.")
	dwdEnabled := flag.Bool("dwd-enabled", false, "Enable DWD provider via Open-Meteo.")
	openWeatherAPIKey := flag.String("openweathermap-api-key", "", "OpenWeatherMap API key. If not set, checks OPENWEATHER_API_KEY env var.")
	yrUserAgent := flag.String("yr-user-agent", "", "User agent for YR provider. If not set, checks USER_AGENT env var.")
	jsonLogs := flag.Bool("json-logs", false, "Use JSON log format instead of text.")

	flag.Var(&cities, "city", "City in format Name:lat,lon[@provider1,provider2]. Repeat this flag for multiple cities.")
	flag.Parse()

	logger := setupLogger(*jsonLogs)

	if len(cities) == 0 {
		logger.Error("at least one --city flag is required")
		os.Exit(1)
	}

	providers := enabledProviders(logger, *dwdEnabled, *openWeatherAPIKey, *yrUserAgent)
	if len(providers) == 0 {
		logger.Error("no providers enabled")
		os.Exit(1)
	}

	if err := validateCityProviders(cities, providers); err != nil {
		logger.Error("invalid city configuration", slog.Any("error", err))
		os.Exit(1)
	}

	providerNames := make([]string, 0, len(providers))
	for _, p := range providers {
		providerNames = append(providerNames, p.Name())
	}
	logger.Info("starting weather-exporter",
		slog.String("providers", strings.Join(providerNames, ", ")),
		slog.Int("cities", len(cities)))

	weatherCollector := collector.NewWeatherCollector(providers, cities, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(collector.RenderPrometheus(weatherCollector.Collect(ctx))))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              *listenAddress,
		Handler:           loggingMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("listening", slog.String("address", *listenAddress))
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	case sig := <-stop:
		logger.Info("shutdown initiated", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("server shutdown failed", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("server stopped gracefully")
	}
}

func setupLogger(jsonFormat bool) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if jsonFormat {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", time.Since(start)))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func enabledProviders(logger *slog.Logger, dwdEnabled bool, openWeatherAPIKey, yrUserAgent string) []provider.Provider {
	enabled := make([]provider.Provider, 0, 3)

	// YR provider: use flag if provided, otherwise fall back to USER_AGENT env var
	userAgent := yrUserAgent
	if userAgent == "" {
		userAgent = os.Getenv("USER_AGENT")
	}
	yr := provider.NewYRProvider(userAgent)
	if err := yr.Init(""); err != nil {
		logger.Warn("provider disabled",
			slog.String("provider", yr.Name()),
			slog.Any("error", err))
	} else {
		enabled = append(enabled, yr)
	}

	if dwdEnabled {
		dwd := provider.NewDWDProvider()
		if err := dwd.Init(""); err != nil {
			logger.Warn("provider disabled",
				slog.String("provider", dwd.Name()),
				slog.Any("error", err))
		} else {
			enabled = append(enabled, dwd)
		}
	}

	// OpenWeatherMap provider: use flag if provided, otherwise fall back to OPENWEATHER_API_KEY env var
	apiKey := openWeatherAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENWEATHER_API_KEY")
	}
	if apiKey != "" {
		openWeather := provider.NewOpenWeatherProvider()
		if err := openWeather.Init(apiKey); err != nil {
			logger.Warn("provider disabled",
				slog.String("provider", openWeather.Name()),
				slog.Any("error", err))
		} else {
			enabled = append(enabled, openWeather)
		}
	} else {
		logger.Info("provider disabled", slog.String("provider", "openweathermap"), slog.String("reason", "neither --openweathermap-api-key flag nor OPENWEATHER_API_KEY env var set"))
	}

	return enabled
}

func providerNames(providers []provider.Provider) []string {
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name())
	}
	return names
}

func validateCityProviders(cities []config.City, providers []provider.Provider) error {
	availableProviders := make(map[string]bool, len(providers))
	for _, name := range providerNames(providers) {
		availableProviders[name] = true
	}

	for _, city := range cities {
		for _, requestedProvider := range city.Providers {
			if !availableProviders[requestedProvider] {
				return fmt.Errorf(
					"city %q requests provider %q which is not available (available: %s)",
					city.Name,
					requestedProvider,
					strings.Join(providerNames(providers), ", "))
			}
		}
	}

	return nil
}
