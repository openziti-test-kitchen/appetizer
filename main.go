package main

import (
	"openziti-test-kitchen/appetizer/underlay"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"

	"openziti-test-kitchen/appetizer/overlay"
)

func main() {
	instanceName := ""
	//logrus.SetLevel(logrus.TraceLevel)
	instanceName = os.Getenv("OPENZITI_DEMO_INSTANCE")
	if strings.TrimSpace(instanceName) == "" {
		hostname, _ := os.Hostname()
		instanceName = hostname
		logrus.Infof("OPENZITI_DEMO_INSTANCE not set. using default of hostname (%s)", hostname)
		// and set it in case it's asked for again...
		_ = os.Setenv("OPENZITI_DEMO_INSTANCE", hostname)
	}

	// special case. if this is 'prod' then set the server identifier to ""
	if strings.ToLower(instanceName) == "prod" {
		logrus.Infof("prod instance detected. using empty string as instanceIdentifier")
		instanceName = ""
		logrus.Infof("instanceName set to: <empty string>")
	} else {
		logrus.Infof("instanceName set to: %s", instanceName)
	}

	topic := underlay.Topic[string]{}
	topic.Start()
	u := underlay.NewUnderlayServer(topic, instanceName)

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

	serverIdentity := u.Prepare("demo-server", recreateNetwork)
	go u.Start()

	go overlay.ServeHTTPOverZiti(serverIdentity, u.HttpServiceName())
	logrus.Infof("started a server listening on the underlay")

	go overlay.StartReflectServer(serverIdentity, u.ReflectServiceName(), topic)
	logrus.Infof("started an OpenZiti reflect server")

	logrus.Infof("servers running. waiting for interrupt")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ch:
		logrus.Infof("signal to shutdown received")
	}
}
