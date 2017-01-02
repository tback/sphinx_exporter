package exporter

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-ini/ini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"

	"github.com/tback/sphinx_exporter/collector"
)

type Config struct {
	// Address to listen on for web interface and telemetry.
	ListenAddress string

	// Path under which to expose metrics.
	MetricPath string

	// Path to .my.cnf file to read MySQL credentials from.",
	ConfigMycnf string

	// Namespace for all metrics.
	Namespace string

	// Subsystem
	Subsystem string

	// Data Source Name
	DSN string
}

// SQL Queries.
const (
	upQuery = `SELECT 1`
)

// Exporter collects Sphinx metrics. It implements prometheus.Collector.
type Exporter struct {
	config          *Config
	dsn             string
	duration, error prometheus.Gauge
	totalScrapes    prometheus.Counter
	scrapeErrors    *prometheus.CounterVec
	sphinxUp        prometheus.Gauge
}

// NewExporter returns a new MySQL exporter for the provided DSN.
func NewExporter(c *Config) *Exporter {
	x := &Exporter{
		config: c,
		dsn:    c.DSN,
		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Namespace,
			Subsystem: c.Subsystem,
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from Sphinx.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Namespace,
			Subsystem: c.Subsystem,
			Name:      "scrapes_total",
			Help:      "Total number of times Sphinx was scraped for metrics.",
		}),
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Namespace,
			Subsystem: c.Subsystem,
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping a Sphinx.",
		}, []string{"collector"}),
		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Namespace,
			Subsystem: c.Subsystem,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from Sphinx resulted in an error (1 for error, 0 for success).",
		}),
		sphinxUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Namespace,
			Name:      "up",
			Help:      "Whether the Sphinx server is up.",
		}),
	}
	if x.dsn == "" {
		var err error
		if x.config.ConfigMycnf == "" {
			log.Fatal("No MySQL DATA_SOURCE_NAME given.")
		}
		x.dsn, err = parseMycnf(x.config.ConfigMycnf)
		if err != nil {
			log.Fatalf("Error loading mycnf: %v", err)
		}
	}
	return x
}

func (e *Exporter) NewDefaultServer() *http.Server {
	prometheus.MustRegister(e)
	mux := http.NewServeMux()
	mux.Handle(e.config.MetricPath, promhttp.Handler())
	s := &http.Server{
		Addr:        e.config.ListenAddress,
		ReadTimeout: 3e9,
		Handler:     mux,
	}

	return s
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	// We cannot know in advance what metrics the exporter will generate
	// from Sphinx. So we use the poor man's describe method: Run a collect
	// and send the descriptors of all the collected metrics. The problem
	// here is that we need to connect to the Sphinx DB. If it is currently
	// unavailable, the descriptors will be incomplete. Since this is a
	// stand-alone exporter and not used as a library within other code
	// implementing additional metrics, the worst that can happen is that we
	// don't detect inconsistent metrics created by this exporter
	// itself. Also, a change in the monitored Sphinx instance may change the
	// exported metrics during the runtime of the exporter.

	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)

	ch <- e.duration
	ch <- e.totalScrapes
	ch <- e.error
	e.scrapeErrors.Collect(ch)
	ch <- e.sphinxUp
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	e.totalScrapes.Inc()
	var err error
	defer func(begun time.Time) {
		e.duration.Set(time.Since(begun).Seconds())
		if err == nil {
			e.error.Set(0)
		} else {
			e.error.Set(1)
		}
	}(time.Now())

	db, err := sql.Open("mysql", e.dsn)
	if err != nil {
		log.Errorln("Error opening connection to database:", err)
		return
	}
	defer db.Close()

	isUpRows, err := db.Query(upQuery)
	if err != nil {
		log.Errorln("Error pinging sphinx:", err)
		e.sphinxUp.Set(0)
		return
	}
	isUpRows.Close()

	e.sphinxUp.Set(1)

	if err = collector.ScrapeStatus(db, ch); err != nil {
		log.Errorln("Error scraping for collect.global_status:", err)
		e.scrapeErrors.WithLabelValues("collect.global_status").Inc()
	}
}

func parseMycnf(config interface{}) (string, error) {
	var dsn string
	cfg, err := ini.Load(config)
	if err != nil {
		return dsn, fmt.Errorf("failed reading ini file: %s", err)
	}
	user := cfg.Section("client").Key("user").String()
	password := cfg.Section("client").Key("password").String()
	if (user == "") || (password == "") {
		return dsn, fmt.Errorf("no user or password specified under [client] in %s", config)
	}
	host := cfg.Section("client").Key("host").MustString("localhost")
	port := cfg.Section("client").Key("port").MustUint(3306)
	socket := cfg.Section("client").Key("socket").String()
	if socket != "" {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/", user, password, socket)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	}
	log.Debugln(dsn)
	return dsn, nil
}

func init() {
	prometheus.MustRegister(version.NewCollector("sphinx_exporter"))
}
