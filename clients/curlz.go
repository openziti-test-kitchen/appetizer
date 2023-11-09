package main

import (
	"github.com/sirupsen/logrus"
	"io"
	"openziti-test-kitchen/appetizer/clients/common"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		logrus.Fatal("Insufficient arguments provided\n\nUsage: ./curlz <serviceName>\n\n")
	}
	url := os.Args[1]
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	idFile := common.GetEnrollmentToken()

	logrus.Infof("Connecting to secure service at: '%s'", url)
	client := common.NewZitifiedHttpClient(idFile)
	resp, err := client.Get(url)
	if err != nil {
		logrus.Fatal(err)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		logrus.Fatal(err)
	}
}
