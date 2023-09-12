package common

import (
	"context"
	"encoding/json"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"strings"
)

type ZitiDialContext struct {
	context ziti.Context
}

func (dc *ZitiDialContext) Dial(_ context.Context, _ string, addr string) (net.Conn, error) {
	service := strings.Split(addr, ":")[0] // will always get passed host:port
	return dc.context.Dial(service)
}

func NewZitiClient(idFile string) *http.Client {
	ctx := ContextFromFile(idFile)
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
			enrollHelper(idFile, resolvedIdFilename)
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

func enrollHelper(jwt string, out string) {
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

	idFilename := filenameWithoutJwtExtension(jwt)

	output, err := os.Create(out)
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
