package main

import (
	"example.com/openzitidemo/manage"
	"example.com/openzitidemo/services"
	"github.com/openziti/edge-api/rest_model"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logrus.Println("Removing demo configuration from " + manage.CtrlAddress)
	manage.DeleteIdentity("demo-server")
	manage.DeleteServicePolicy("demo-server-bind")
	manage.DeleteServicePolicy("demo-client-dial")
	manage.DeleteService("reflectService")
	manage.DeleteService("httpService")

	logrus.Println("Adding demo configuration to " + manage.CtrlAddress)
	manage.CreateService("reflectService", "demo-services")
	manage.CreateService("httpService", "demo-services")
	manage.CreateServicePolicy("demo-client-dial", rest_model.DialBindDial, rest_model.Roles{"#demo.clients"}, rest_model.Roles{"#demo-services"})
	manage.CreateServicePolicy("demo-server-bind", rest_model.DialBindBind, rest_model.Roles{"#demo.servers"}, rest_model.Roles{"#demo-services"})
	_ = manage.CreateIdentity(rest_model.IdentityTypeDevice, "demo-server", "demo.servers")
	time.Sleep(time.Second)
	serverIdentity := manage.EnrollIdentity("demo-server")

	go manage.ServeHTTP(18000)

	go services.ServeHTTP(serverIdentity)
	logrus.Println("Started an OpenZiti reflect server")

	go services.Server(serverIdentity, "reflectService")
	logrus.Println("Started an OpenZiti reflect server")

	logrus.Println("Servers running. Waiting for interrupt")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ch:
		logrus.Println("Signal to shutdown received")
	}
}
