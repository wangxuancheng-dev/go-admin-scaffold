package types

import "time"

// UserSearchFilters represents search filters for user queries
type UserSearchFilters struct {
	Username string
	Email    string
	Status   *int // pointer to allow nil (no filter)
	RoleID   uint
}

// UserExportFilters represents filter criteria for exporting users
type UserExportFilters struct {
	Username  string
	Email     string
	Status    *int
	StartTime time.Time
	EndTime   time.Time
}
