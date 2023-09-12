package manage

import (
	"encoding/base64"
	"fmt"
	"github.com/openziti/edge-api/rest_model"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"net/http"
	"text/template"
)

func ServeHTTP(port int) {
	listener := CreateUnderlayListener(port)
	svr := &http.Server{}
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(serveIndexHTML))
	mux.Handle("/add-me-to-openziti", http.HandlerFunc(addToOpenZiti))
	mux.Handle("/download-token", http.HandlerFunc(downloadToken))
	svr.Handler = mux
	if err := svr.Serve(listener); err != nil {
		logrus.Fatal(err)
	}
}

func serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./index.html")
}

func addToOpenZiti(w http.ResponseWriter, r *http.Request) {
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

	DeleteIdentity(name)
	createdIdentity := CreateIdentity(rest_model.IdentityTypeUser, name, "demo.clients")

	tmpl, err := template.ParseFiles("add-to-openziti-response.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Token string
		Name  string
	}{
		Token: createdIdentity.Payload.Data.ID,
		Name:  name,
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func downloadToken(w http.ResponseWriter, r *http.Request) {
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
		return "", fmt.Errorf("Length must be greater than zero")
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

func CreateUnderlayListener(port int) net.Listener {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Printf("Started an insecure server on %d\n", port)
	return ln
}
