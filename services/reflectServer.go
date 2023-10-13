package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	goaway "github.com/TwiN/go-away"
	"github.com/openziti/sdk-golang/ziti/edge"
	"io/ioutil"
	"net/http"
	"openziti-test-kitchen/appetizer/clients/common"
	"openziti-test-kitchen/appetizer/manage"
	"os"
	"strings"

	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"

	"github.com/microcosm-cc/bluemonday"
)

type ReflectServer struct {
	topic            manage.Topic[string]
	classifierClient *http.Client
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

	newClassifierClient := common.NewZitiClientFromContext(ctx)
	r := &ReflectServer{
		topic:            topic,
		classifierClient: newClassifierClient,
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
	p := bluemonday.UGCPolicy()

	//line delimited
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			logrus.Error(err)
			break
		}
		logrus.Info("about to read a string :")
		logrus.Infof("                  read : %s", strings.TrimSpace(line))

		var resp string
		if goaway.IsProfane(line) {
			resp = fmt.Sprintf("please remember to be kind and keep it clean. not sending your message. you sent me: %s", line)
		} else {
			//let it through
			isOffensive := r.IsOffensive(line)
			logrus.Infof("Verifying the line is not offensive: %t, %s", isOffensive, line)
			if isOffensive {
				resp = fmt.Sprintf("Your message seems like it might be offensive. We didn't relay it. you sent me: %s", line)
			} else {
				// ACTUALLY let it through
				resp = fmt.Sprintf("you sent me: %s", line)
				r.topic.Notify(fmt.Sprintf("event: notify\n"))
				html := p.Sanitize(line)
				r.topic.Notify(fmt.Sprintf("data: %s:%s\n\n", conn.SourceIdentifier(), html))
			}
		}
		i++
		_, _ = rw.WriteString(resp)
		_ = rw.Flush()
		logrus.Infof("       responding with : %s", strings.TrimSpace(resp))
	}
}

type ClassifierBody struct {
	Text string `json:"text"`
}
type ClassifierResult struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

func (r ReflectServer) IsOffensive(input string) bool {

	url := "http://classifier-service:80/api/v1/classify"

	logrus.Infof("Classifying input as offensive at: '%s'", url)
	inputBody := ClassifierBody{
		Text: input,
	}

	jsonData, _ := json.Marshal(inputBody)
	reader := bytes.NewBuffer(jsonData)

	resp, err := r.classifierClient.Post(url, "application/json", reader)
	if err != nil {
		logrus.Error(err)
		return false
	}

	// Read the response body into a byte slice
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		logrus.Error(readErr)
		return false
	}

	// Create an instance of the struct to unmarshal into
	var results []ClassifierResult

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &results)
	if err != nil {
		logrus.Error(readErr)
		return false
	}
	result := results[0]
	return result.Label == "Offensive"
}
