package overlay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	goaway "github.com/TwiN/go-away"
	"github.com/openziti/sdk-golang/ziti/edge"
	"io/ioutil"
	"net"
	"net/http"
	"openziti-test-kitchen/appetizer/clients/common"
	"openziti-test-kitchen/appetizer/underlay"
	"os"
	"strings"
	"time"

	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"

	"github.com/microcosm-cc/bluemonday"
)

type OffensiveResult int

const (
	NOT_OFFENSIVE OffensiveResult = iota
	COULD_NOT_CLASSIFY
	OFFENSIVE
)

type ReflectServer struct {
	topic            underlay.Topic[string]
	classifierClient *http.Client
	zitiCtx          ziti.Context
	mattermostClient *http.Client
	mattermostUrl    string
}

func StartReflectServer(zitiCfg *ziti.Config, serviceName string, topic underlay.Topic[string]) {
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
		logrus.Warnf("could not load identity from environment OPENZITI_IDENTITY. will not attempt to send messages to mattermost: %v", jsonErr)
	} else {
		cfg, err := ziti.NewContext(&c)
		if err != nil {
			logrus.Warnf("error when loading identity specified in environment OPENZITI_IDENTITY. will not attempt to send messages to mattermost: %v", err)
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

func (r ReflectServer) serve(listener edge.Listener) {
	logrus.Infof("ready to accept connections")
	for {
		conn, _ := listener.AcceptEdge()
		go r.accept(conn)
	}
}

func (r ReflectServer) accept(conn edge.Conn) {
	if conn == nil {
		logrus.Fatal("connection is nil!")
	}

	logrus.Infof("accepted connection from %s", conn.SourceIdentifier())
	defer func() {
		logrus.Infof("closing connection for %s", conn.SourceIdentifier())
		_ = conn.Close()
	}()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	rw := bufio.NewReadWriter(reader, writer)

	i := 0
	p := bluemonday.StrictPolicy()

	//line delimited
	for {
		duration := 60 * time.Second
		buffer := make([]byte, 1024) //1k max
		line, err := readLineWithTimeout(conn, duration, buffer)
		if err != nil {
			var netErr net.Error
			ok := errors.As(err, &netErr)
			if ok && netErr.Timeout() {
				logrus.Infof("%s idle for longer than timeout (%s)", conn.SourceIdentifier(), duration)
				return
			} else if err != nil {
				logrus.Error(err)
			}
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
			logrus.Infof("verifying the line is not offensive: %t, %s", isOffensive != 0, line)

			ma := &MattermostAttachment{
				Text: line,
			}
			relayMessage := false
			switch isOffensive {
			case OFFENSIVE:
				resp = fmt.Sprintf("Your message seems like it might be offensive. We didn't relay it. you sent me: %s", line)
				ma.ThumbUrl = offensiveZiggy
				ma.Color = "#FF0000"
				ma.Pretext = "A message classified as offensive has been received. Is it actually offensive? "
				addPollAction(ma)
			case NOT_OFFENSIVE:
				// ACTUALLY let it through
				ma.ThumbUrl = coolZiggy
				ma.Color = "#00FF00"
				ma.Pretext = "A new message was received: "
				resp = fmt.Sprintf("you sent me: %s", line)
				relayMessage = true
			case COULD_NOT_CLASSIFY:
				ma.ThumbUrl = questionZiggy
				ma.Color = "#FFBF00"
				ma.Pretext = "A new message was received but could not be qualified for offensiveness: "
				resp = fmt.Sprintf("you sent a message, but it can't be qualified at this time for offensiveness: %s", line)
				relayMessage = true
			}
			if relayMessage {
				r.topic.Notify(fmt.Sprintf("event: notify\n"))
				html := p.Sanitize(line)
				source := conn.SourceIdentifier()
				if strings.ContainsAny(source, "@") {
					//strip out anything after the @...
					parts := strings.Split(source, "@")
					source = parts[0]
				}
				r.topic.Notify(fmt.Sprintf("data: %s:%s\n\n", source, html))
			}
			r.notifyMattermost(ma, conn.SourceIdentifier())
		}
		i++
		_, _ = rw.WriteString(resp)
		_, _ = rw.WriteString("\n")
		_ = rw.Flush()
		logrus.Infof("       responding with : %s", strings.TrimSpace(resp))
	}
}

func readLineWithTimeout(conn net.Conn, duration time.Duration, buff []byte) (string, error) {
	// Create a buffered reader
	reader := bufio.NewReader(conn)
	// Set a timeout for reading a line
	_ = conn.SetReadDeadline(time.Now().Add(duration))

	n, err := reader.Read(buff)
	if err != nil {
		return "", err
	}

	// Find the position of the first newline character
	newlineIndex := -1
	for i := 0; i < n; i++ {
		if buff[i] == '\n' {
			newlineIndex = i
			break
		}
	}

	// If a newline is found, discard bytes after the newline
	if newlineIndex != -1 {
		n = newlineIndex + 1
	}

	// Convert the buffer to a string
	line := string(buff[:n])

	if len(line) == len(buff) {
		line += "\n" //add a newline if we hit the limit
	}
	return line, nil
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

func (r ReflectServer) IsOffensive(input string) OffensiveResult {
	url := "http://classifier-service:80/api/v1/classify"

	logrus.Infof("trying to classify input as offensive: '%s'", url)
	inputBody := ClassifierBody{
		Text: input,
	}

	jsonData, _ := json.Marshal(inputBody)
	reader := bytes.NewBuffer(jsonData)

	resp, err := r.classifierClient.Post(url, "application/json", reader)
	if err != nil {
		if strings.ContainsAny(err.Error(), "has no term") {
			logrus.Warnf("seems like the classifier overlay is down. can't classify input [%s]: %v", input, err)
		} else {
			logrus.Warnf("could not classify input, unknown error. input:[%s]. error: %v", input, err)
		}
		return COULD_NOT_CLASSIFY
	}
	// Read the response body into a byte slice
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		logrus.Error(readErr)
		return NOT_OFFENSIVE
	}

	// Create an instance of the struct to unmarshal into
	var results []ClassifierResult

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &results)
	if err != nil {
		logrus.Error(readErr)
		return NOT_OFFENSIVE
	}
	result := results[0]
	if result.Label == "Offensive" {
		return OFFENSIVE
	} else {
		return NOT_OFFENSIVE
	}
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

var offensiveZiggy = "https://raw.githubusercontent.com/openziti/branding/main/images/ziggy/closeups/Ziggy-Angry-Closeup.png"
var coolZiggy = "https://raw.githubusercontent.com/openziti/branding/main/images/ziggy/closeups/Ziggy-Cool-Closeup.png"
var appetizerZiggy = "https://raw.githubusercontent.com/openziti/branding/main/images/ziggy/closeups/Ziggy-Chef-Closeup.png"
var questionZiggy = "https://raw.githubusercontent.com/openziti/branding/main/images/ziggy/closeups/Ziggy-has-a-Question-Closeup.png"

func (r ReflectServer) notifyMattermost(ma *MattermostAttachment, from string) {
	if r.mattermostClient != nil && r.mattermostUrl != "" {
		m := MattermostHook{
			Attachments: []MattermostAttachment{*ma},
			IconUrl:     &appetizerZiggy,
			Username:    &from,
		}
		jsonData, _ := json.Marshal(m)
		bodyReader := bytes.NewBuffer(jsonData)
		_, err := r.mattermostClient.Post(r.mattermostUrl, "application/json", bodyReader)
		if err != nil {
			logrus.Errorf("error when posting message to mattermost: %v", err)
		}
	} else {
		logrus.Infof("mattermost not configured. Skipping mattermost send.")
	}
}
