package backer

import (
	"regexp"
)

var (
	// C contains the global configuration loaded from config file.
	// This is intentional — config is loaded once at startup and never modified concurrently.
	C Config

	// excludePatterns contains compiled regex patterns from C.ExcludePatterns.
	// Populated by LoadConfig, used by isExcluded during directory walking.
	excludePatterns []*regexp.Regexp
)

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
