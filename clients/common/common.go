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
	"path/filepath"
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
			logrus.Infof("using existing file: %s", resolvedIdFilename)
		} else {
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

func findFiles(root, prefix, suffix string) ([]string, error) {
	var matchingFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file matches the pattern
		if strings.HasSuffix(info.Name(), suffix) && !info.IsDir() {
			if strings.HasPrefix(info.Name(), prefix) {
				matchingFiles = append(matchingFiles, path)
			}
		}

		// If the current path is not the root, skip subdirectories
		if path != root && info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	})

	return matchingFiles, err
}

func appetizerUrl() string {
	appetizer := os.Getenv("OPENZITI_APPETIZER_URL")
	if appetizer == "" {
		appetizer = DEFAULT_APPETIZER_URL
	}
	return appetizer
}

func GetEnrollmentToken() string {
	ctrl := appetizerUrl()

	// Find all files matching the pattern in the directory
	matchingFiles, err := findFiles(".", PrefixedName("randomizer_"), "json")
	if err != nil {
		logrus.Fatalf("error: %s", err)
	}

	if len(matchingFiles) > 1 {
		logrus.Fatalf("too many files found matching randomizer_*.json, delete the incorrect file(s)")
	}
	if len(matchingFiles) == 1 {
		logrus.Infof("appetizer unfinished. using existing identity file: %s", matchingFiles[0])
		return matchingFiles[0]
	}

	logrus.Infof("no identity file found, ordering a new one")
	newIdUrl := ctrl + "/sample"
	resp, err := http.Get(newIdUrl)
	if err != nil {
		logrus.Fatal("cannot connect to controller at " + newIdUrl)
	}
	if resp.StatusCode > 299 {
		logrus.Fatal("there was a problem getting a token from: " + newIdUrl)
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

	logrus.Infof("serving and enrolling identity file: %s", filename)
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
	logrus.Infof("strong identity successfully written to: %s", idFilename)
	_ = os.Remove(jwt)
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

// PrefixedName is a function that accounts for multiple instances of the appetizer service running
// against the same OpenZiti overlay through the OPENZITI_DEMO_INSTANCE env var
func PrefixedName(input string) string {
	meta := appetizerUrl() + "/meta"
	resp, err := http.Get(meta)
	if err != nil {
		logrus.Fatalf("could not get metadata from %s", meta)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Check if the status code is successful (2xx)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var result map[string]string
		err := json.Unmarshal(body, &result)
		if err != nil {
			logrus.Warnf("could not parse metadata? %v", err)
			return ""
		}

		qualifier := result["qualifier"]
		if qualifier != "" {
			return qualifier + "_" + input
		}
		return input
	}
	logrus.Warnf("error: HTTP %d - %s\n", resp.StatusCode, resp.Status)
	return input
}
