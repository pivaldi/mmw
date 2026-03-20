package application_test

import (
	"github.com/pivaldi/mmw/auth/internal/application"
	defauth "github.com/pivaldi/mmw/contracts/definitions/auth"
)

// Ensure AuthApplicationService satisfies the defauth.AuthService contract.
var _ defauth.AuthService = (*application.AuthApplicationService)(nil)
