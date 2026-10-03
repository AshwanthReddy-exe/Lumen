//go:build !osusergo

package main

import (
	"errors"
	"os/user"
	"path/filepath"
)

const accountHomeLookupUsesSystemRecord = true

func canonicalAccountHome() (string, error) {
	account, err := user.Current()
	if err != nil {
		return "", err
	}
	if account.HomeDir == "" || !filepath.IsAbs(account.HomeDir) {
		return "", errors.New("account home must be an absolute path")
	}
	return filepath.Clean(account.HomeDir), nil
}
