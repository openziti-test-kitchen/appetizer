package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"openziti-test-kitchen/appetizer/clients/common"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		logrus.Fatal("insufficient arguments provided\n\nUsage: ./reflect <serviceName> [optional:identityFile]\n\n")
	}
	serviceName := common.PrefixedName(os.Args[1])

	var idFile string
	if len(os.Args) > 2 {
		idFile = os.Args[2]
	} else {
		idFile = common.GetEnrollmentToken()
	}
	logrus.Infof("serving identity file: %s", idFile)

	ctx := common.ContextFromFile(idFile)

	foundSvc, ok := ctx.GetService(serviceName)
	if !ok {
		log.Fatalf("service name [%s] was not found?", serviceName)
	}
	logrus.Infof("found service named: %s", *foundSvc.Name)

	svc, err := ctx.Dial(serviceName) //dial the service using the given name
	if err != nil {
		log.Fatalf("error when dialing service name %s. %v", serviceName, err)
	}
	logrus.Infof("connected to %s successfully.", serviceName)
	logrus.Info("you may now type a line to be sent to the server (press enter to send)")
	logrus.Info("the line will be sent to the reflect server and returned")

	reader := bufio.NewReader(os.Stdin) //setup a reader for reading input from the commandline
	conRead := bufio.NewReader(svc)
	conWrite := bufio.NewWriter(svc)

	for {
		fmt.Print("Enter some text to send: ")
		text, err := reader.ReadString('\n') //read a line from input
		if err != nil {
			fmt.Println(err)
			return // exit the program when it reads EOF
		}
		bytesRead, err := conWrite.WriteString(text)
		_ = conWrite.Flush()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("wrote", bytesRead, "bytes")
		}
		fmt.Print("Sent    :", text)
		read, err := conRead.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return // exit the program when it reads EOF
		} else {
			fmt.Println("Received:", read)
		}
	}
}
