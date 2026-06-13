package cmdb

import (
	"fmt"
	"regexp"
	"strings"
)

var namingSegmentPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`)

func ValidateStandardAppID(value string) (ApplicationNaming, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) != 5 {
		return ApplicationNaming{}, fmt.Errorf("%w: app_id must use organization.businessDomain.capabilityDomain.application.role", ErrInvalidAppID)
	}

	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return ApplicationNaming{}, fmt.Errorf("%w: segment %d is empty", ErrInvalidAppID, index+1)
		}
		if !namingSegmentPattern.MatchString(part) {
			return ApplicationNaming{}, fmt.Errorf("%w: segment %d %q must use lower-case kebab text", ErrInvalidAppID, index+1, part)
		}
		parts[index] = part
	}

	return ApplicationNaming{
		Organization:     parts[0],
		BusinessDomain:   parts[1],
		CapabilityDomain: parts[2],
		Application:      parts[3],
		Role:             parts[4],
	}, nil
}

func StandardAppIDFromRequest(appID string, appCode string) string {
	appID = strings.TrimSpace(appID)
	if appID != "" {
		return appID
	}
	return strings.TrimSpace(appCode)
}
