package validate

import (
	"github.com/go-playground/validator/v10"
)

// Default is a shared validator for structs outside Gin binding (services, jobs, CLI).
var Default = validator.New()
