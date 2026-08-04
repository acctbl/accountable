package secret

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	ProviderFile             = "file"
	ProviderInfisical        = "infisical"
	AuthAWSIAM               = "aws_iam"
	InfisicalCloudEUEndpoint = "https://eu.infisical.com"
)

var secretRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

// Ref is an opaque lookup key. It never contains secret material.
type Ref string

type Config struct {
	Provider          string
	Directory         string
	SiteURL           string
	AWSRegion         string
	ProjectID         string
	Environment       string
	SecretPath        string
	AuthMethod        string
	MachineIdentityID string
}

func ParseRef(value string) (Ref, error) {
	if !secretRefPattern.MatchString(value) || filepath.IsAbs(value) || value == "." || value == ".." {
		return "", errors.New("must be a 1 to 128 character reference using only letters, digits, '.', '_', ':', '/', or '-'")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", errors.New("must not traverse directories")
		}
	}
	if filepath.Clean(value) != value || value == ".." || len(value) >= 3 && value[:3] == "../" {
		return "", errors.New("must not traverse directories")
	}
	return Ref(value), nil
}
