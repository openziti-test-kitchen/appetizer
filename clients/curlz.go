package main

import (
	"example.com/openzitidemo/clients/common"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Insufficient arguments provided\n\nUsage: ./curlz <serviceName> <identityFile>\n\n")
		return
	}
	url := os.Args[1]
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}

	resp, err := common.NewZitiClient(os.Args[2]).Get(url)
	if err != nil {
		logrus.Fatal(err)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		logrus.Fatal(err)
	}
}
