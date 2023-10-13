package manage

import (
	"encoding/base64"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"text/template"
	"time"
)

type UnderlayServer struct {
	topic              Topic[string]
	instanceIdentifier string
}

func NewUnderlayServer(topic Topic[string], instanceIdentifier string) UnderlayServer {
	return UnderlayServer{
		topic:              topic,
		instanceIdentifier: instanceIdentifier,
	}
}

func (u UnderlayServer) Prepare(forceRecreate bool) *ziti.Config {
	logrus.Println("Removing demo configuration from " + CtrlAddress)
	hostname, _ := os.Hostname()
	svrId := u.scopedName("demo-server-" + hostname)
	reflectSvcName := u.ReflectServiceName()
	svcAttrName := u.scopedName("demo-services")
	httpSvcName := u.HttpServiceName()
	bindSp := u.scopedName("demo-server-bind")
	bindSpRole := u.scopedName("demo.servers")
	dialSp := u.scopedName("demo-server-dial")
	dialSpRole := u.scopedName("demo.clients")
	DeleteIdentity(svrId)
	if forceRecreate {
		DeleteServicePolicy(bindSp)
		DeleteServicePolicy(dialSp)
		DeleteService(reflectSvcName)
		DeleteService(httpSvcName)
	}

	logrus.Println("Adding demo configuration to " + CtrlAddress)
	CreateService(reflectSvcName, svcAttrName)
	CreateService(httpSvcName, svcAttrName)
	CreateServicePolicy(dialSp, rest_model.DialBindDial, rest_model.Roles{"#" + dialSpRole}, rest_model.Roles{"#" + svcAttrName})
	CreateServicePolicy(bindSp, rest_model.DialBindBind, rest_model.Roles{"#" + bindSpRole}, rest_model.Roles{"#" + svcAttrName})
	bindAttributes := &rest_model.Attributes{bindSpRole, "classifier-clients"}
	_ = CreateIdentity(rest_model.IdentityTypeDevice, svrId, bindAttributes)
	time.Sleep(time.Second)
	return EnrollIdentity(svrId)
}

func (u UnderlayServer) HttpServiceName() string {
	return u.scopedName("httpService")
}
func (u UnderlayServer) ReflectServiceName() string {
	return u.scopedName("reflectService")
}

func (u UnderlayServer) Start() {
	mux := http.NewServeMux()
	mux.Handle("/add-me-to-openziti", http.HandlerFunc(u.addToOpenZiti))
	mux.Handle("/download-token", http.HandlerFunc(u.downloadToken))
	mux.Handle("/sse", http.HandlerFunc(u.sse))
	mux.Handle("/messages", http.HandlerFunc(u.messagesHandler))
	mux.Handle("/", http.FileServer(http.Dir("http_content")))

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print the working directory
	fmt.Println("Current Working Directory:", wd)

	var svr *http.Server
	if DomainName != "" {
		certmagic.DefaultACME.Agreed = true
		email := os.Getenv("OPENZITI_ACME_EMAIL")
		certmagic.DefaultACME.Email = email
		ca := os.Getenv("OPENZITI_CA")
		if ca != "prod" {
			logrus.Info("Using LetsEncryptStagingCA - not prod")
			certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
		} else {
			logrus.Info("Using LetsEncryptProductionCA!!! Don't abuse the rate limit")
			certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
		}

		err := certmagic.HTTPS([]string{DomainName}, mux)
		if err != nil {
			log.Fatalf("Failed to create https: %v", err)
		}
		ln, err := certmagic.Listen([]string{DomainName})
		if err != nil {
			log.Fatalf("Failed to create listener: %v", err)
		}
		tlsConfig, err := certmagic.TLS([]string{DomainName})
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

func (u UnderlayServer) serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./index.html")
}
func (u UnderlayServer) serveOverview(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./overview.png")
}
func (u UnderlayServer) messagesHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./messages.html")
}

func (u UnderlayServer) addToOpenZiti(w http.ResponseWriter, r *http.Request) {
	var name string
	if r.URL.Query().Get("randomizer") != "" {
		randomId, _ := generateRandomID(8)
		name = "randomizer_" + randomId
	} else {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad Request: Your request is invalid.", http.StatusBadRequest)
			return
		}

		name = r.Form.Get("name")
		logrus.Printf("Received name: %s", name)
	}

	if name == "" {
		http.Error(w, "Invalid input. name form field not provided", http.StatusBadRequest)
		return
	}

	name = u.scopedName(name)

	DeleteIdentity(name)
	createdIdentity := CreateIdentity(rest_model.IdentityTypeUser, name, &rest_model.Attributes{u.scopedName("demo.clients")})

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

func (u UnderlayServer) scopedName(name string) string {
	if u.instanceIdentifier == "" {
		return name
	} else {
		return u.instanceIdentifier + "_" + name
	}
}

func (u UnderlayServer) downloadToken(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("token")
	if t == "" {
		http.Error(w, "Token not available", http.StatusBadRequest)
		return
	}

	id := FindIdentityDetail(t)
	jwtToken := id.Data.Enrollment.Ott.JWT
	if jwtToken == "" {
		http.Error(w, "Token not available", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+*id.Data.Name+".jwt")
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(jwtToken))
}

func generateRandomID(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than zero")
	}

	// Determine how many random bytes we need
	numBytes := (length * 6) / 8 // 6 bits per character for base64 encoding

	// Generate random bytes
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes as a base64 string
	randomID := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Trim the string to the desired length
	return randomID[:length], nil
}

func (u UnderlayServer) sse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	id, _ := generateRandomID(10)
	te := u.topic.NewEntry(id)

	for {
		select {
		case msg := <-te.Messages: //<-time.After(1 * time.Second):
			_, _ = fmt.Fprintf(w, "%s", msg)
			w.(http.Flusher).Flush() // Flush the response to the client
		case <-r.Context().Done():
			u.topic.RemoveReceiver(te)
			fmt.Println("Client closed connection.")
			return
		}
	}
}
