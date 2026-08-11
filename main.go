package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hown3d/renovate-apk-indexer/pkg/apk"
	"github.com/hown3d/renovate-apk-indexer/pkg/renovate"
)

const mainIndex = "https://dl-cdn.alpinelinux.org/{alpineVersion}/main/{architecture}/APKINDEX.tar.gz"

var (
	updateInterval = flag.Int("update-interval", 4, "update interval of the apk package index in hours")
	apkIndexUrls   = flag.String("apk-index-url", mainIndex, "comma-separated URLs of the apk indexes to get the package information from")
	logLevel       = new(slog.Level)
	logOutput      = flag.String("log-output", "text", "representation for logs (text,json)")
)

func main() {
	flag.TextVar(logLevel, "log-level", slog.LevelInfo, "log level")
	flag.Parse()

	var l *slog.Logger
	logOpts := &slog.HandlerOptions{
		Level: logLevel,
	}

	switch *logOutput {
	case "json":
		jsonHandler := slog.NewJSONHandler(os.Stdout, logOpts)
		l = slog.New(jsonHandler)
	case "text":
		textHandler := slog.NewTextHandler(os.Stdout, logOpts)
		l = slog.New(textHandler)
	}
	slog.SetDefault(l)

	urls := strings.Split(*apkIndexUrls, ",")
	apkContext := apk.New(http.DefaultClient, urls)
	apkPackagesPerAlpineVersionAndArchitecture, errors := apkContext.GetApkPackages()
	if len(errors) != 0 {

		slog.Error("error updating apk packages", "err", errors)
	}

	ticker := time.NewTicker(time.Duration(*updateInterval) * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				slog.Info("updating apk packages")
				var err error
				apkPackagesPerAlpineVersionAndArchitecture, errors = apkContext.GetApkPackages()
				if len(errors) != 0 {
					slog.Error("error updating apk packages", "err", err)
					continue
				}
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", healthHandler)
	mux.HandleFunc("/livez", healthHandler)
	mux.HandleFunc("/apk/{alpineVersion}/{architecture}/{package}", apkHandler(apkContext, apkPackagesPerAlpineVersionAndArchitecture))

	slog.Info("serving on :3000")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		slog.Error("serving http ", "err", err)
		os.Exit(1)
	}
}

func apkHandler(apkContext apk.Context, apkPackagesPerAlpineVersionAndArchitecture *apk.PackagePerAlpineVersionAndArchitecture) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		alpineVersion := r.PathValue("alpineVersion")
		architecture := r.PathValue("architecture")
		packageName := r.PathValue("package")
		err := apkContext.AddAlpineVersionAndArchitecture(alpineVersion, architecture)
		if err != nil {
			slog.Error("error adding version and architecture", "alpineVersion", alpineVersion, "architectures", architecture, "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "internal server error")
			return
		}
		//fmt.Printf("%v\n", apkPackagesPerAlpineVersionAndArchitecture)
		packages, ok := (*apkPackagesPerAlpineVersionAndArchitecture)[alpineVersion][architecture][packageName]
		if !ok {
			slog.Debug("package not found", "packageName", packageName)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "%s not found in apkIndex", packageName)
			return
		}

		slog.Debug("packages found", "packageName", packageName)
		datasource := renovate.TransformAPKPackage(packages)
		if err := json.NewEncoder(w).Encode(datasource); err != nil {
			slog.Error("encoding datasource", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "internal server error")
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("ok"))
}
