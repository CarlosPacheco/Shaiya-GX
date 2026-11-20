package config

import (
	"fmt"
	"os"
	"strconv"
)

// NetworkPolicy is immutable for a connection. Startup and acceptance validate
// the same bounded overrides; a malformed value never silently disables a cap.
type NetworkPolicy struct {
	Connections, PerIP, IPAttempts, AccountAttempts, InputFrames, InputBytes, OutputPackets, OutputBytes int
	Establishments, Economy                                                                              int
}

func Network(service string) (NetworkPolicy, error) {
	p := NetworkPolicy{Establishments: 120, Economy: 20, Connections: 1200, PerIP: 32, IPAttempts: 30, AccountAttempts: 10, InputFrames: 512, InputBytes: 1 << 20, OutputPackets: 256, OutputBytes: 1 << 20}
	key := "RAPTURE_WORLD_MAX_CONNECTIONS"
	if service == "login" {
		p.Connections = 200
		key = "RAPTURE_LOGIN_MAX_CONNECTIONS"
	}
	values := []struct {
		key    string
		target *int
		max    int
	}{{"RAPTURE_CONNECTIONS_PER_IP_PER_MINUTE", &p.Establishments, 10000}, {"RAPTURE_ECONOMY_ACTIONS_PER_SECOND", &p.Economy, 10000}, {key, &p.Connections, 10000}, {"RAPTURE_CONNECTIONS_PER_IP", &p.PerIP, 10000}, {"RAPTURE_LOGIN_ATTEMPTS_PER_IP", &p.IPAttempts, 10000}, {"RAPTURE_LOGIN_ATTEMPTS_PER_ACCOUNT", &p.AccountAttempts, 10000}, {"RAPTURE_INPUT_FRAMES_PER_SECOND", &p.InputFrames, 65536}, {"RAPTURE_INPUT_BYTES_PER_SECOND", &p.InputBytes, 64 << 20}, {"RAPTURE_OUTPUT_PACKETS", &p.OutputPackets, 65536}, {"RAPTURE_OUTPUT_BYTES", &p.OutputBytes, 64 << 20}}
	for _, value := range values {
		if raw, ok := os.LookupEnv(value.key); ok {
			number, err := strconv.Atoi(raw)
			if err != nil || number < 1 || number > value.max {
				return p, fmt.Errorf("%s must be 1..%d", value.key, value.max)
			}
			*value.target = number
		}
	}
	return p, nil
}
