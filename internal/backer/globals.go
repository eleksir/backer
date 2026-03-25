package backer

import (
	"regexp"
)

var (
	// C contains the global configuration loaded from config file.
	C Config
	// excludePatterns contains compiled regex patterns from C.ExcludePatterns.
	excludePatterns []*regexp.Regexp
)

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
