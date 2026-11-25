package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type config struct {
	port                   int
	monolithURL            *url.URL
	moviesServiceURL       *url.URL
	eventsServiceURL       *url.URL
	gradualMigration       bool
	moviesMigrationPercent int
}

func loadConfig() (config, error) {
	var cfg config

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil || port < 1 || port > 65535 {
		return config{}, fmt.Errorf("PORT must be an integer between 1 and 65535")
	}
	cfg.port = port

	for _, upstream := range []struct {
		env    string
		target **url.URL
	}{
		{"MONOLITH_URL", &cfg.monolithURL},
		{"MOVIES_SERVICE_URL", &cfg.moviesServiceURL},
		{"EVENTS_SERVICE_URL", &cfg.eventsServiceURL},
	} {
		target, err := url.Parse(os.Getenv(upstream.env))
		if err != nil || (target.Scheme != "http" && target.Scheme != "https") ||
			target.Hostname() == "" || target.User != nil || target.ForceQuery ||
			target.RawQuery != "" || target.Fragment != "" {
			return config{}, fmt.Errorf("%s must be an absolute HTTP(S) URL without credentials, query, or fragment", upstream.env)
		}
		if port := target.Port(); port != "" {
			number, err := strconv.Atoi(port)
			if err != nil || number < 1 || number > 65535 {
				return config{}, fmt.Errorf("%s must use a port between 1 and 65535", upstream.env)
			}
		}
		*upstream.target = target
	}

	cfg.gradualMigration, err = strconv.ParseBool(os.Getenv("GRADUAL_MIGRATION"))
	if err != nil {
		return config{}, fmt.Errorf("GRADUAL_MIGRATION must be a boolean")
	}

	cfg.moviesMigrationPercent, err = strconv.Atoi(os.Getenv("MOVIES_MIGRATION_PERCENT"))
	if err != nil || cfg.moviesMigrationPercent < 0 || cfg.moviesMigrationPercent > 100 {
		return config{}, fmt.Errorf("MOVIES_MIGRATION_PERCENT must be an integer between 0 and 100")
	}

	return cfg, nil
}
