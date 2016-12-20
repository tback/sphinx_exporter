// Scrape `SHOW GLOBAL STATUS`.

package collector

import (
	"database/sql"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query
	globalStatusQuery = `SHOW STATUS`
	// Subsytem.
	globalStatus = "status"
)

var commandDesc = prometheus.NewDesc(
	prometheus.BuildFQName(namespace, globalStatus, "command"),
	"Commands", []string{"command"}, nil,
)

var descs = map[string]*prometheus.Desc{
	"uptime": prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "uptime"),
		"Uptime of Sphinx Server.",
		nil, nil,
	),
	"connections" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "connections"),
		"Connections to Sphinx Server.", nil, nil,
	),
	"maxed_out" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "maxed_out"),
		"Rejected Queries.", nil, nil,
	),
	"command_search" : commandDesc,
	"command_excerpt" : commandDesc,
	"command_update" : commandDesc,
	"command_delete" : commandDesc,
	"command_keywords" : commandDesc,
	"command_persist" : commandDesc,
	"command_status" : commandDesc,
	"command_flushattrs" : commandDesc,
	"agent_connect" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "agent_connect"),
		"Agent Connect.", nil, nil,
	),
	"agent_retry" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "agent_retry"),
		"Agent Retry.", nil, nil,
	),
	"queries" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "queries"),
		"Queries.", nil, nil,
	),
	"dist_queries" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "dist_queries"),
		"Distributed Queries.", nil, nil,
	),
	"query_wall" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "query_wall"),
		"Wall Time spent on queries.", nil, nil,
	),
	"query_cpu" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "query_cpu"),
		"CPU Time spent on queries.", nil, nil,
	),
	"dist_wall" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "dist_wall"),
		"Wall Time Spent on distributed Queries.", nil, nil,
	),
	"dist_local" : prometheus.NewDesc(prometheus.BuildFQName(
		namespace, globalStatus, "dist_local"),
		"Time spent on dist local queries.", nil, nil,
	),
	"dist_wait" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "dist_wait"),
		"Total waiting time on agents.", nil, nil,
	),
	"query_reads" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "query_reads"),
		"Query reads.", nil, nil,
	),
	"query_readkb" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "query_readkb"),
		"Query read KB.", nil, nil,
	),
	"query_readtime" : prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "query_readtime"),
		"Query read time.", nil, nil,
	),
}

// ScrapeGlobalStatus collects from `SHOW GLOBAL STATUS`.
func ScrapeStatus(db *sql.DB, ch chan<- prometheus.Metric) error {
	globalStatusRows, err := db.Query(globalStatusQuery)
	if err != nil {
		return err
	}
	defer globalStatusRows.Close()

	var key string
	var val sql.RawBytes

	for globalStatusRows.Next() {
		if err := globalStatusRows.Scan(&key, &val); err != nil {
			return err
		}
		if floatVal, ok := parseStatus(val); ok {
			// Unparsable values are silently skipped.
			key = strings.ToLower(key)
			if strings.HasPrefix(key, "command_"){
				label := key[8:]
				ch <- prometheus.MustNewConstMetric(
					descs[key], prometheus.CounterValue, floatVal, label,
				)
				continue
			}
			desc, ok := descs[key]
			if ok {
				ch <- prometheus.MustNewConstMetric(
					desc, prometheus.CounterValue, floatVal,
				)
			}
		}
	}

	return nil
}
