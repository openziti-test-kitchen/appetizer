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

	var offset = 0
	var idFile string
	if len(os.Args) > 5 {
		idFile = os.Args[2]
	} else {
		// offset by -1 since an identityFile is not provided, others need to be shifted
		offset = -1
		idFile = common.GetEnrollmentToken()
	}
	logrus.Infof("serving identity file: %s", idFile)
	input1 := os.Args[3+offset]
	operator := os.Args[4+offset]
	input2 := os.Args[5+offset]

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
