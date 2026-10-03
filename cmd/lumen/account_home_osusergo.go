//go:build osusergo

package main

import "errors"

const accountHomeLookupUsesSystemRecord = false

func canonicalAccountHome() (string, error) {
	return "", errors.New("LaunchAgent setup requires system account home lookup")
}
