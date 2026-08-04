package database

import (
	"time"

	"github.com/acctbl/accountable/internal/platform/secret"
)

const (
	TLSDisable    = "disable"
	TLSVerifyFull = "verify-full"
)

type Config struct {
	Host                string
	Port                uint16
	Name                string
	User                string
	Role                string
	PasswordRef         secret.Ref
	TLSMode             string
	TLSRootCAFile       string
	ConnectTimeout      time.Duration
	StatementTimeout    time.Duration
	HealthCheckInterval time.Duration
	MaxConnections      int32
	Timezone            string
}
