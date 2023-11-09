package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"openziti-test-kitchen/appetizer/clients/common"
	"os"
)

func main() {
	serviceName := os.Args[1]

	idFile := ""
	if len(os.Args) < 2 {
		idFile = common.GetEnrollmentToken()
	} else {
		idFile = os.Args[2]
	}
	ctx := common.ContextFromFile(idFile)

	foundSvc, ok := ctx.GetService(serviceName)
	if !ok {
		panic("error when retrieving all the overlay for the provided config")
	}
	logrus.Infof("found service named: %s", *foundSvc.Name)

	svc, err := ctx.Dial(serviceName) //dial the service using the given name
	if err != nil {
		panic(fmt.Sprintf("error when dialing service name %s. %v", serviceName, err))
	}
	logrus.Infof("Connected to %s successfully.", serviceName)
	logrus.Info("You may now type a line to be sent to the server (press enter to send)")
	logrus.Info("The line will be sent to the reflect server and returned")

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
