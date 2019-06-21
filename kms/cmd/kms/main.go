package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	keymanager "github.com/offen/offen/kms/keymanager/local"
	"github.com/offen/offen/kms/router"
	"github.com/sirupsen/logrus"
)

func main() {
	var (
		port     = flag.String("port", os.Getenv("PORT"), "the port the server binds to")
		logLevel = flag.String("level", "info", "the application's log level")
		origin   = flag.String("origin", "http://localhost:9977", "the CORS origin")
	)
	flag.Parse()

	logger := logrus.New()
	parsedLogLevel, parseErr := logrus.ParseLevel(*logLevel)
	if parseErr != nil {
		logger.WithError(parseErr).Fatalf("unable to parse given log level %s", *logLevel)
	}
	logger.SetLevel(parsedLogLevel)

	manager, err := keymanager.New(func() ([]byte, error) {
		keyFile := os.Getenv("KEY_FILE")
		return ioutil.ReadFile(keyFile)
	})

	if err != nil {
		logger.WithError(err).Fatal("error setting up keymanager")
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", *port),
		Handler: router.New(*origin, manager, logger),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	logger.Infof("KMS server now listening on port %s.", *port)
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err.Error())
	}
}
