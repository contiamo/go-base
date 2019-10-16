package middleware

import (
	"fmt"
	"os"
)

var hostname = ""

func getHostname() string {
	if len(hostname) == 0 {
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			_ = fmt.Errorf("unable to retrieve hostname - setting to unknown")
			hostname = "unknown"
		}
	}
	return hostname
}
