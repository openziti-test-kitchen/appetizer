package common

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"
)

const DEFAULT_APPETIZER_URL = "https://appetizer.openziti.io"

type ZitiDialContext struct {
	context ziti.Context
}

func (dc *ZitiDialContext) Dial(_ context.Context, _ string, addr string) (net.Conn, error) {
	service := strings.Split(addr, ":")[0] // will always get passed host:port
	return dc.context.Dial(service)
}

func NewZitifiedHttpClient(idFile string) *http.Client {
	ctx := ContextFromFile(idFile)
	return NewZitiClientFromContext(ctx)
}
func NewZitiClientFromContext(ctx ziti.Context) *http.Client {
	zitiDialContext := ZitiDialContext{context: ctx}

	zitiTransport := http.DefaultTransport.(*http.Transport).Clone() // copy default transport
	zitiTransport.DialContext = zitiDialContext.Dial
	zitiTransport.TLSClientConfig.InsecureSkipVerify = true
	return &http.Client{Transport: zitiTransport}
}

func ContextFromFile(idFile string) ziti.Context {
	resolvedIdFilename := idFile
	if strings.HasSuffix(idFile, ".json") {
		// just use it as the identity file...
	} else if strings.HasSuffix(idFile, ".jwt") {
		// might need to enroll it if not enrolled already
		resolvedIdFilename = filenameWithoutJwtExtension(idFile) + ".json"
		if _, err := os.Stat(resolvedIdFilename); err == nil {
			logrus.Infof("Using existing file: %s", resolvedIdFilename)
		} else {
			logrus.Infof("First time using %s. Automatically enrolling to %s", idFile, resolvedIdFilename)
			enrollHelper(idFile)
		}
	}

	cfg, err := ziti.NewConfigFromFile(resolvedIdFilename)
	if err != nil {
		logrus.Fatal(err)
	}

	ctx, err := ziti.NewContext(cfg)
	if err != nil {
		logrus.Fatal(err)
	}
	return ctx
}

func GetEnrollmentToken() string {
	// Scan current directory for random*.jwt (reuse random generated id)
	//    Scan for random .json file
	// TODO: Add messages telling user a new identity is being created, also when reusing .jwt
	if len(os.Args) > 2 {
		return os.Args[2]
	}

	ctrl := os.Getenv("OPENZITI_APPETIZER_URL")
	if ctrl == "" {
		ctrl = DEFAULT_APPETIZER_URL
	}
	// TODO: make paths constants
	newIdUrl := ctrl + "/sample"
	resp, err := http.Get(newIdUrl)
	if err != nil {
		logrus.Fatal("cannot connect to controller at " + newIdUrl)
	}
	filename := getFilenameFromHeader(resp.Header)
	if filename == "" {
		// If Content-Disposition is not present or doesn't contain a filename, use a default name
		filename = "downloaded_file.txt"
	}
	file, err := os.Create(filename)
	if err != nil {
		logrus.Fatal("cannot create file: " + filename)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		logrus.Fatal(err)
	}

	return filename
}

func getFilenameFromHeader(header http.Header) string {
	contentDisposition := header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func enrollHelper(jwt string) {
	raw, err := os.ReadFile(jwt)
	if err != nil {
		logrus.Fatal(err)
	}
	tkn, _, err := enroll.ParseToken(string(raw))
	if err != nil {
		logrus.Fatal(err)
	}
	flags := enroll.EnrollmentFlags{
		Token:  tkn,
		KeyAlg: "RSA",
	}
	conf, err := enroll.Enroll(flags)
	if err != nil {
		logrus.Fatal(err)
	}

	idFilename := filenameWithoutJwtExtension(jwt) + ".json"

	output, err := os.Create(idFilename)
	if err != nil {
		logrus.Fatalf("failed to open file '%s': %s", idFilename, err.Error())
	}
	defer func() { _ = output.Close() }()

	enc := json.NewEncoder(output)
	enc.SetEscapeHTML(false)
	encErr := enc.Encode(&conf)

	if encErr != nil {
		logrus.Fatalf("enrollment successful but the identity file was not able to be written to: %s [%s]", idFilename, encErr)
	}
	logrus.Infof("enrolled successfully. identity file written to: %s", idFilename)
}

func filenameWithoutJwtExtension(jwt string) string {
	if strings.HasSuffix(jwt, ".jwt") {
		return jwt[:len(jwt)-len(".jwt")]
	} else {
		//doesn't end with .jwt - so just slap a .json on the end and call it a day
		return jwt
	}
}

func GetRandomName() string {
	randomId, _ := GenerateRandomID(8)
	return "randomizer_" + randomId
}

func GenerateRandomID(length int) (string, error) {
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
