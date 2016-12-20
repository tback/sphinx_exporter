package collector

import (
	"database/sql"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Exporter namespace.
	namespace = "sphinx"
	// Math constant for picoseconds to seconds.
	picoSeconds = 1e12
)

var logRE = regexp.MustCompile(`.+\.(\d+)$`)

func newDesc(subsystem, name, help string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, name),
		help, nil, nil,
	)
}

func parseStatus(data sql.RawBytes) (float64, bool) {
	stat := string(data)
	switch stat {
	case "Yes", "ON":
		return 1, true
	case "No", "OFF":
		return 0, true
	case "Connecting":
		// SHOW SLAVE STATUS Slave_IO_Running can return "Connecting" which is a non-running state.
		return 0, true
	case "Primary":
		// SHOW GLOBAL STATUS like 'wsrep_cluster_status' can return "Primary" or "Non-Primary"/"Disconnected"
		return 1, true
	case "Non-Primary", "Disconnected":
		return 0, true
	}

	if logNum := logRE.Find(data); logNum != nil {
		value, err := strconv.ParseFloat(string(logNum), 64)
		return value, err == nil
	}

	value, err := strconv.ParseFloat(stat, 64)
	return value, err == nil
}
