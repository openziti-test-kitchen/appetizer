package services

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"
)

func Server(zitiCfg *ziti.Config, serviceName string) {
	ctx, err := ziti.NewContext(zitiCfg)

	if err != nil {
		logrus.Fatal(err)
	}

	listener, err := ctx.Listen(serviceName)
	if err != nil {
		logrus.Fatal(err)
	}
	serve(listener)

	sig := make(chan os.Signal)
	s := <-sig
	logrus.Infof("received %s: shutting down...", s)
}

func serve(listener net.Listener) {
	logrus.Infof("ready to accept connections")
	for {
		conn, _ := listener.Accept()
		logrus.Infof("new connection accepted")
		go accept(conn)
	}
}

func accept(conn net.Conn) {
	if conn == nil {
		logrus.Fatal("connection is nil!")
	}
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	rw := bufio.NewReadWriter(reader, writer)
	//line delimited
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			logrus.Error(err)
			break
		}
		logrus.Info("about to read a string :")
		logrus.Infof("                  read : %s", strings.TrimSpace(line))
		resp := fmt.Sprintf("you sent me: %s", line)
		_, _ = rw.WriteString(resp)
		_ = rw.Flush()
		logrus.Infof("       responding with : %s", strings.TrimSpace(resp))
	}
}
