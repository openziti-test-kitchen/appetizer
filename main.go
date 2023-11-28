package main

import (
	"openziti-test-kitchen/appetizer/underlay"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"

	"openziti-test-kitchen/appetizer/overlay"
)

func main() {

	isProd := os.Getenv("OPENZITI_IS_PROD")
	instanceName := ""
	//logrus.SetLevel(logrus.TraceLevel)
	if isProd != "y" {
		instanceName = os.Getenv("OPENZITI_DEMO_INSTANCE")

		if instanceName == "" {
			hostname, _ := os.Hostname()
			instanceName = hostname
			logrus.Infof("OPENZITI_DEMO_INSTANCE not set. using default of hostname (%s)", hostname)
		}
	}
	recreateNetworkEnv := os.Getenv("OPENZITI_RECREATE_NETWORK")
	var recreateNetwork bool
	if recreateNetworkEnv == "" {
		recreateNetwork = true
	} else {
		b, boolParseErr := strconv.ParseBool(recreateNetworkEnv)
		if boolParseErr != nil {
			recreateNetwork = false
		} else {
			recreateNetwork = b
		}
	}

	topic := underlay.Topic[string]{}
	topic.Start()

	u := underlay.NewUnderlayServer(topic, instanceName)
	serverIdentity := u.Prepare(recreateNetwork)
	go u.Start()

	go overlay.ServeHTTPOverZiti(serverIdentity, u.HttpServiceName())
	logrus.Println("Started a server listening on the underlay")

	go overlay.StartReflectServer(serverIdentity, u.ReflectServiceName(), topic)
	logrus.Println("Started an OpenZiti reflect server")

	logrus.Println("Servers running. Waiting for interrupt")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ch:
		logrus.Println("Signal to shutdown received")
	}
}
