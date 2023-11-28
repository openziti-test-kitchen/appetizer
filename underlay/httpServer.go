package underlay

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"
	"log"
	"net"
	"net/http"
	"openziti-test-kitchen/appetizer/clients/common"
	"openziti-test-kitchen/appetizer/manage"
	"os"
	"strings"
	"text/template"
	"time"
)

type Server struct {
	topic              Topic[string]
	instanceIdentifier string
}

func NewUnderlayServer(topic Topic[string], instanceIdentifier string) Server {
	return Server{
		topic:              topic,
		instanceIdentifier: instanceIdentifier,
	}
}

func (u Server) Prepare(forceRecreate bool) *ziti.Config {
	logrus.Println("removing demo configuration from " + manage.CtrlAddress)
	svrId := u.scopedName("demo-server")
	reflectSvcName := u.ReflectServiceName()
	svcAttrName := u.scopedName("demo-services")
	httpSvcName := u.HttpServiceName()
	bindSp := u.scopedName("demo-server-bind")
	bindSpRole := u.scopedName("demo.servers")
	dialSp := u.scopedName("demo-server-dial")
	dialSpRole := u.scopedName("demo.clients")
	manage.DeleteIdentity(svrId)
	if forceRecreate {
		manage.DeleteServicePolicy(bindSp)
		manage.DeleteServicePolicy(dialSp)
		manage.DeleteService(reflectSvcName)
		manage.DeleteService(httpSvcName)
	}

	logrus.Infof("adding demo configuration to %s for identity %s", manage.CtrlAddress, svrId)
	manage.CreateService(reflectSvcName, svcAttrName)
	manage.CreateService(httpSvcName, svcAttrName)
	manage.CreateServicePolicy(dialSp, rest_model.DialBindDial, rest_model.Roles{"#" + dialSpRole}, rest_model.Roles{"#" + svcAttrName})
	manage.CreateServicePolicy(bindSp, rest_model.DialBindBind, rest_model.Roles{"#" + bindSpRole}, rest_model.Roles{"#" + svcAttrName})
	bindAttributes := &rest_model.Attributes{bindSpRole, "classifier-clients"}
	_ = manage.CreateIdentity(rest_model.IdentityTypeDevice, svrId, bindAttributes)
	time.Sleep(time.Second)
	return manage.EnrollIdentity(svrId)
}

func (u Server) HttpServiceName() string {
	return u.scopedName("httpService")
}
func (u Server) ReflectServiceName() string {
	return u.scopedName("reflectService")
}

func (u Server) Start() {
	mux := http.NewServeMux()
	mux.Handle("/add-me-to-openziti", http.HandlerFunc(u.addToOpenZiti))
	mux.Handle("/taste", http.HandlerFunc(u.addToOpenZiti))
	mux.Handle("/download-token", http.HandlerFunc(u.downloadToken))
	mux.Handle("/sse", http.HandlerFunc(u.sse))
	mux.Handle("/messages", http.HandlerFunc(u.messagesHandler))
	mux.Handle("/getinvite", http.HandlerFunc(u.inviteHandler))
	mux.Handle("/sample", http.HandlerFunc(u.sample))
	mux.Handle("/meta", http.HandlerFunc(u.meta))
	mux.Handle("/", http.FileServer(http.Dir("http_content")))

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		logrus.Warnf("could not get working directory. %v", err)
		return
	}

	// Print the working directory
	logrus.Infof("current working directory: %s", wd)

	var svr *http.Server
	if manage.DomainName != "" {
		certmagic.DefaultACME.Agreed = true
		email := os.Getenv("OPENZITI_ACME_EMAIL")
		certmagic.DefaultACME.Email = email
		ca := os.Getenv("OPENZITI_CA")
		if ca != "prod" {
			logrus.Info("using LetsEncryptStagingCA - not prod")
			certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
		} else {
			logrus.Info("using LetsEncryptProductionCA!!! don't abuse the rate limit")
			certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
		}

		err := certmagic.HTTPS([]string{manage.DomainName}, mux)
		if err != nil {
			log.Fatalf("Failed to create https: %v", err)
		}
		ln, err := certmagic.Listen([]string{manage.DomainName})
		if err != nil {
			log.Fatalf("Failed to create listener: %v", err)
		}
		tlsConfig, err := certmagic.TLS([]string{manage.DomainName})
		if err != nil {
			log.Fatalf("Failed to create TLS: %v", err)
		}
		svr = &http.Server{
			TLSConfig: tlsConfig,
		}
		svr.Handler = mux
		if err := svr.ServeTLS(ln, "", ""); err != nil {
			logrus.Fatal(err)
		}
	} else {
		svr = &http.Server{}
		svr.Handler = mux
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", 18000))
		if err != nil {
			logrus.Fatal(err)
		}
		if err := svr.Serve(ln); err != nil {
			logrus.Fatal(err)
		}
	}
}

func (u Server) serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./index.html")
}
func (u Server) serveOverview(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./overview.png")
}
func (u Server) messagesHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./messages.html")
}
func (u Server) inviteHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("http_content/invite.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	whoFromUser := r.URL.Query().Get("who")
	who := whoFromUser + suf
	if who == "" {
		http.Error(w, "Bad Request: Your request is invalid.", http.StatusBadRequest)
		return
	}
	inputBytes := []byte(who)
	who64 := base64.RawURLEncoding.EncodeToString(inputBytes)

	link := fmt.Sprintf("https://%s/taste?ziti=%s", r.Host, who64)

	if err != nil {
		logrus.Warnf("input [%s] could not be base64 encoded? %v", who, err)
		http.Error(w, "Bad Request: Your request is invalid.", http.StatusBadRequest)
		return
	}
	data := struct {
		Who  string
		Link string
	}{
		Who:  whoFromUser,
		Link: link,
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

const suf = "_taste"

func (u Server) addToOpenZiti(w http.ResponseWriter, r *http.Request) {
	var name string
	taster := r.URL.Query().Get("taste")
	if taster == "" {
		taster = r.URL.Query().Get("ziti")
	}
	if taster != "" {
		decodedBytes, err := base64.RawStdEncoding.DecodeString(taster)
		if err != nil {
			logrus.Warnf("input [%s] could not be base64 decoded: %v", taster, err)
			http.Error(w, "Bad Request: Your request is invalid.", http.StatusBadRequest)
			return
		}
		name = strings.TrimSpace(string(decodedBytes))
		// do just the most minimal amount of checksum'ing... nothing fancy at all...
		if !strings.HasSuffix(name, suf) {
			logrus.Warnf("input [%s] was base64 encoded but didn't contain the expected suffix [%s] [%s]: %v", taster, suf, name, err)
			http.Error(w, "Bad Request: Your request is invalid.", http.StatusBadRequest)
			return
		}
		name = strings.Replace(name, suf, "", -1)
		logrus.Infof("we have a new taster: %s", name)
	} else if r.URL.Query().Get("randomizer") != "" {
		name = common.GetRandomName()
	} else {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad Request: Your request is invalid.", http.StatusBadRequest)
			return
		}

		name = r.Form.Get("name")
		logrus.Printf("received name: %s", name)
	}

	if name == "" {
		http.Error(w, "Invalid input. name form field not provided", http.StatusBadRequest)
		return
	}

	name = u.scopedName(name)

	manage.DeleteIdentity(name)
	createdIdentity := manage.CreateIdentity(rest_model.IdentityTypeUser, name, &rest_model.Attributes{u.scopedName("demo.clients")})

	tmpl, err := template.ParseFiles("http_content/add-to-openziti-response.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Token      string
		Name       string
		HttpSvc    string
		ReflectSvc string
	}{
		Token:      createdIdentity.Payload.Data.ID,
		Name:       name,
		HttpSvc:    u.HttpServiceName(),
		ReflectSvc: u.ReflectServiceName(),
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (u Server) scopedName(name string) string {
	if u.instanceIdentifier == "" {
		return name
	} else {
		return u.instanceIdentifier + "_" + name
	}
}

func (u Server) downloadToken(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("token")
	if t == "" {
		http.Error(w, "Token not available", http.StatusBadRequest)
		return
	}

	id := manage.FindIdentityDetail(t)
	jwtToken := id.Data.Enrollment.Ott.JWT
	if jwtToken == "" {
		http.Error(w, "Token not available", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+*id.Data.Name+".jwt")
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(jwtToken))
}

func (u Server) sse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	id, _ := common.GenerateRandomID(10)
	te := u.topic.NewEntry(id)

	for {
		select {
		case msg := <-te.Messages: //<-time.After(1 * time.Second):
			_, _ = fmt.Fprintf(w, "%s", msg)
			w.(http.Flusher).Flush() // Flush the response to the client
		case <-r.Context().Done():
			u.topic.RemoveReceiver(te)
			logrus.Debug("client closed connection.")
			return
		}
	}
}

func (u Server) sample(w http.ResponseWriter, r *http.Request) {
	name := u.scopedName(common.GetRandomName())

	manage.DeleteIdentity(name)
	createdIdentity := manage.CreateIdentity(rest_model.IdentityTypeUser, name, &rest_model.Attributes{u.scopedName("demo.clients")})

	t := createdIdentity.Payload.Data.ID
	if t == "" {
		http.Error(w, "Token not available", http.StatusBadRequest)
		return
	}

	id := manage.FindIdentityDetail(t)
	jwtToken := id.Data.Enrollment.Ott.JWT
	if jwtToken == "" {
		http.Error(w, "Token not available", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+*id.Data.Name+".jwt")
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(jwtToken))
}

func (u Server) meta(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"qualifier": os.Getenv("OPENZITI_DEMO_INSTANCE")}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(response)
}
