package seeders

import (
	"fmt"

	"app/internal/database/seeder"
)

var globalManager *seeder.SeederManager

type pendingEntry struct {
	name string
	s    *seeder.Seeder
}

var pendingRegistrations []pendingEntry

// Register registers a seeder with the global manager or stores it temporarily (FIFO init order).
func Register(name string, s *seeder.Seeder) {
	if globalManager != nil {
		globalManager.Register(name, s)
		return
	}
	for _, p := range pendingRegistrations {
		if p.name == name {
			panic(fmt.Sprintf("seeder %s already registered", name))
		}
	}
	pendingRegistrations = append(pendingRegistrations, pendingEntry{name: name, s: s})
}

// SetGlobalManager sets the global seeder manager and registers all pending seeders
func SetGlobalManager(manager *seeder.SeederManager) {
	globalManager = manager

	for _, p := range pendingRegistrations {
		manager.Register(p.name, p.s)
	}

	pendingRegistrations = nil
}
