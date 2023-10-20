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
	zitiCtx          ziti.Context
	mattermostClient *http.Client
	mattermostUrl    string
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

	ozId := os.Getenv("OPENZITI_IDENTITY")
	c := ziti.Config{}
	jsonErr := json.Unmarshal([]byte(ozId), &c)
	if jsonErr != nil {
		logrus.Warnf("could not load identity from environment OPENZITI_IDENTITY. Will not attempt to send messages to mattermost: %v", jsonErr)
	} else {
		cfg, err := ziti.NewContext(&c)
		if err != nil {
			logrus.Warnf("error when loading identity specified in environment OPENZITI_IDENTITY. Will not attempt to send messages to mattermost: %v", err)
		} else {
			r.zitiCtx = cfg
			r.mattermostClient = common.NewZitiClientFromContext(r.zitiCtx)
			r.mattermostUrl = os.Getenv("OPENZITI_MATTERMOST_URL")
		}
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
	p := bluemonday.StrictPolicy()

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

			ma := &MattermostAttachment{
				Text: line,
			}
			if isOffensive {
				resp = fmt.Sprintf("Your message seems like it might be offensive. We didn't relay it. you sent me: %s", line)
				ma.ThumbUrl = offensiveZiggy
				ma.Color = "#FF0000"
				ma.Pretext = "A message classified as offensive has been received. Is it actually offensive? "
				addPollAction(ma)
			} else {
				// ACTUALLY let it through
				ma.ThumbUrl = coolZiggy
				ma.Color = "#00FF00"
				ma.Pretext = "A new message was received: "
				resp = fmt.Sprintf("you sent me: %s", line)
				r.topic.Notify(fmt.Sprintf("event: notify\n"))
				html := p.Sanitize(line)
				r.topic.Notify(fmt.Sprintf("data: %s:%s\n\n", conn.SourceIdentifier(), html))
			}
			r.sendMessage(ma, conn.SourceIdentifier())
		}
		i++
		_, _ = rw.WriteString(resp)
		_ = rw.Flush()
		logrus.Infof("       responding with : %s", strings.TrimSpace(resp))
	}
}

var yes = MattermostAction{
	Id:    "voteYes",
	Type:  "button",
	Name:  "Yes",
	Style: "danger",
}
var no = MattermostAction{
	Id:    "voteNo",
	Type:  "button",
	Name:  "No",
	Style: "success",
}

func addPollAction(a *MattermostAttachment) {
	a.MattermostActions = []MattermostAction{yes, no}
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

type MattermostContext struct {
	Action string `json:"action"`
}
type MattermostIntegration struct {
	Url     string `json:"url"`
	Context MattermostContext
}
type MattermostAction struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Style       string `json:"style"`
	Integration MattermostIntegration
}

type MattermostAttachment struct {
	ThumbUrl          string             `json:"thumb_url"`
	Text              string             `json:"text"`
	AuthorName        string             `json:"author_name"`
	Color             string             `json:"color"`
	Pretext           string             `json:"pretext"`
	MattermostActions []MattermostAction `json:"actions"`
}

type MattermostHook struct {
	Channel     *string                `json:"channel"`
	Username    *string                `json:"username"`
	IconUrl     *string                `json:"icon_url"`
	IconEmoji   *string                `json:"icon_emoji"`
	Attachments []MattermostAttachment `json:"attachments"`
	Type        *string                `json:"Type"`
	Props       *string                `json:"props"`
}

var offensiveZiggy = "https://raw.githubusercontent.com/openziti/branding/main/images/ziggy/closeups/Ziggy-Angry-Closeup.png?raw=true"
var coolZiggy = "https://github.com/openziti/branding/blob/main/images/ziggy/closeups/Ziggy-Cool-Closeup.png?raw=true"
var appetizerZiggy = "https://github.com/openziti/branding/blob/main/images/ziggy/closeups/Ziggy-Chef-Closeup.png?raw=true"

func (r ReflectServer) sendMessage(ma *MattermostAttachment, from string) {
	if r.mattermostClient != nil {
		m := MattermostHook{
			Attachments: []MattermostAttachment{*ma},
			IconUrl:     &appetizerZiggy,
			Username:    &from,
		}
		jsonData, _ := json.Marshal(m)
		bodyReader := bytes.NewBuffer(jsonData)
		resp, err := r.mattermostClient.Post(r.mattermostUrl, "application/json", bodyReader)
		if err != nil {
			logrus.Errorf("error when posting message to mattermost: %v", err)
		}
		logrus.Infof("response from mattermost: %v", resp)
	} else {
		logrus.Infof("Mattermost not configured. Skipping mattermost send.")
	}
}
