package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"

	exporter "github.com/tback/sphinx_exporter"
)

var (
	config = &exporter.Config{
		Namespace: "sphinx",
		Subsystem: "exporter",
	}
	showVersion *bool
)

func init() {
	showVersion = flag.Bool(
		"version", false,
		"Print version information.",
	)
	flag.StringVar(&config.ListenAddress,
		"web.listen-address", ":9104",
		"Address to listen on for web interface and telemetry.",
	)
	flag.StringVar(&config.MetricPath,
		"web.telemetry-path", "/metrics",
		"Path under which to expose metrics.",
	)
	flag.StringVar(&config.ConfigMycnf,
		"config.my-cnf", path.Join(os.Getenv("HOME"), ".my.cnf"),
		"Path to .my.cnf file to read MySQL credentials from.",
	)
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("sphinx_exporter"))
		os.Exit(0)
	}

	log.Infoln("Starting sphinx_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	if dsn := os.Getenv("DATA_SOURCE_NAME"); dsn != "" {
		config.DSN = dsn
	}

	srv := exporter.NewExporter(config).NewDefaultServer()

	log.Infoln("Listening on", config.ListenAddress)
	log.Fatal(srv.ListenAndServe())
}
