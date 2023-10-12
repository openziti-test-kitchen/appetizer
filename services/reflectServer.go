package services

import (
	"bufio"
	"fmt"
	goaway "github.com/TwiN/go-away"
	"github.com/openziti/sdk-golang/ziti/edge"
	"openziti-test-kitchen/appetizer/manage"
	"os"
	"strings"

	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"

	_ "github.com/TwiN/go-away"
)

type ReflectServer struct {
	topic manage.Topic[string]
}

func StartReflectServer(zitiCfg *ziti.Config, serviceName string, topic manage.Topic[string]) {
	ctx, err := ziti.NewContext(zitiCfg)
	if err != nil {
		logrus.Fatal(err)
	}

	listener, err := ctx.Listen(serviceName)
	if err != nil {
		logrus.Fatal(err)
	}

	r := &ReflectServer{
		topic: topic,
	}

	r.serve(listener)

	sig := make(chan os.Signal)
	s := <-sig
	logrus.Infof("received %s: shutting down...", s)
}

func (r *ReflectServer) serve(listener edge.Listener) {
	logrus.Infof("ready to accept connections")
	for {
		conn, _ := listener.AcceptEdge()
		logrus.Infof("new connection accepted")
		go r.accept(conn)
	}
}

func (r *ReflectServer) accept(conn edge.Conn) {
	if conn == nil {
		logrus.Fatal("connection is nil!")
	}
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	rw := bufio.NewReadWriter(reader, writer)

	i := 0
	//line delimited
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			logrus.Error(err)
			break
		}
		logrus.Info("about to read a string :")
		logrus.Infof("                  read : %s", strings.TrimSpace(line))
		r.topic.Notify(fmt.Sprintf("event: notify\n"))
		resp := fmt.Sprintf("you sent me: %s", line)
		if goaway.IsProfane(line) {
			resp += "... but please remember to be kind and keep it clean"
		}
		r.topic.Notify(fmt.Sprintf("data: %s:%s\n\n", conn.SourceIdentifier(), line))
		i++
		_, _ = rw.WriteString(resp)
		_ = rw.Flush()
		logrus.Infof("       responding with : %s", strings.TrimSpace(resp))
	}
}
