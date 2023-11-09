package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	neturl "net/url"
	"openziti-test-kitchen/appetizer/clients/common"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Printf("Insufficient arguments provided\n\nUsage: ./math <serviceName> [optional:identityFile] input1 operator input2\n\n")
		return
	}
	url := os.Args[1]
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}

	var idFile string
	var input1 string
	var operator string
	var input2 string
	if len(os.Args) > 6 {
		input1 = os.Args[3]
		operator = os.Args[4]
		input2 = os.Args[5]
		idFile = os.Args[2]
	} else {
		input1 = os.Args[2]
		operator = os.Args[3]
		input2 = os.Args[4]
		idFile = common.GetEnrollmentToken()
		logrus.Infof("identity file not provided, using identity file: %s", idFile)
	}

	url = fmt.Sprintf("%s/domath?input1=%s&operator=%s&input2=%s", url, input1, neturl.QueryEscape(operator), input2)

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
