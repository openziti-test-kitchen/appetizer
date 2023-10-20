package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"

	"openziti-test-kitchen/appetizer/manage"
	"openziti-test-kitchen/appetizer/services"
)

func main() {
	//logrus.SetLevel(logrus.TraceLevel)
	instanceName := os.Getenv("OPENZITI_DEMO_INSTANCE")
	if instanceName == "" {
		hostname, _ := os.Hostname()
		instanceName = hostname
		logrus.Infof("OPENZITI_DEMO_INSTANCE not set. using default of hostname (%s)", hostname)
	}
	recreateNetworkEnv := os.Getenv("OPENZITI_RECREATE_NETWORK")
	var recreateNetwork bool
	if recreateNetworkEnv == "" {
		recreateNetwork = false
	} else {
		b, boolParseErr := strconv.ParseBool(recreateNetworkEnv)
		if boolParseErr != nil {
			recreateNetwork = false
		} else {
			recreateNetwork = b
		}
	}

	topic := manage.Topic[string]{}
	topic.Start()

	u := manage.NewUnderlayServer(topic, instanceName)
	serverIdentity := u.Prepare(recreateNetwork)
	go u.Start()

	go services.ServeHTTPOverZiti(serverIdentity, u.HttpServiceName())
	logrus.Println("Started a server listening on the underlay")

	go services.StartReflectServer(serverIdentity, u.ReflectServiceName(), topic)
	logrus.Println("Started an OpenZiti reflect server")

	logrus.Println("Servers running. Waiting for interrupt")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ch:
		logrus.Println("Signal to shutdown received")
	}
}
