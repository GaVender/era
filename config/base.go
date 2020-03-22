package config

import "time"

var (
	env          string
	timeLocation time.Location
)

func Env() string {
	return env
}

func TimeLocation() time.Location {
	return timeLocation
}
