package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openziti/edge-api/rest_model"
	"github.com/sirupsen/logrus"

	"openziti-test-kitchen/appetizer/manage"
	"openziti-test-kitchen/appetizer/services"
)

func main() {
	manage.DemoInstanceName = os.Getenv("OPENZITI_DEMO_INSTANCE")

	if manage.DemoInstanceName == "" {
		hostname, _ := os.Hostname()
		manage.DemoInstanceName = hostname
		logrus.Infof("OPENZITI_DEMO_INSTANCE not set. using default of hostname (%s)", hostname)
	}

	logrus.Println("Removing demo configuration from " + manage.CtrlAddress)
	svrId := manage.DemoInstanceName + "_demo-server"
	reflectSvcName := manage.DemoInstanceName + "_reflectService"
	svcAttrName := manage.DemoInstanceName + "_demo-services"
	httpSvcName := manage.DemoInstanceName + "_httpService"
	bindSp := manage.DemoInstanceName + "_demo-server-bind"
	bindSpRole := manage.DemoInstanceName + "_demo.servers"
	dialSp := manage.DemoInstanceName + "_demo-server-dial"
	dialSpRole := manage.DemoInstanceName + "_demo.clients"
	manage.DeleteIdentity(svrId)
	manage.DeleteServicePolicy(bindSp)
	manage.DeleteServicePolicy(dialSp)
	manage.DeleteService(reflectSvcName)
	manage.DeleteService(httpSvcName)

	logrus.Println("Adding demo configuration to " + manage.CtrlAddress)
	manage.CreateService(reflectSvcName, svcAttrName)
	manage.CreateService(httpSvcName, svcAttrName)
	manage.CreateServicePolicy(dialSp, rest_model.DialBindDial, rest_model.Roles{"#" + dialSpRole}, rest_model.Roles{"#" + svcAttrName})
	manage.CreateServicePolicy(bindSp, rest_model.DialBindBind, rest_model.Roles{"#" + bindSpRole}, rest_model.Roles{"#" + svcAttrName})
	_ = manage.CreateIdentity(rest_model.IdentityTypeDevice, svrId, bindSpRole)
	time.Sleep(time.Second)
	serverIdentity := manage.EnrollIdentity(svrId)

	topic := manage.Topic[string]{}
	topic.Start()

	go manage.StartUnderlayServer(topic)

	go services.ServeHTTPOverZiti(serverIdentity, httpSvcName)
	logrus.Println("Started a server listening on the underlay")

	go services.StartReflectServer(serverIdentity, reflectSvcName, topic)
	logrus.Println("Started an OpenZiti reflect server")

	logrus.Println("Servers running. Waiting for interrupt")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ch:
		logrus.Println("Signal to shutdown received")
	}
}
