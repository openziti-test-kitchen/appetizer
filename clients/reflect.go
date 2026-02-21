package main

import (
	"bufio"
	"fmt"
	"log"
	"openziti-test-kitchen/appetizer/clients/common"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
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

	ctx := common.ContextFromFile(idFile)

	foundSvc, ok := ctx.GetService(serviceName)
	if !ok {
		log.Fatalf("service name [%s] was not found?", serviceName)
	}
	logrus.Debugf("found service named: %s", *foundSvc.Name)

	svc, err := ctx.Dial(serviceName) //dial the service using the given name
	if err != nil {
		log.Fatalf("error when dialing service name %s. %v", serviceName, err)
	}
	logrus.Infof("end to end encrypted connection to %s established", serviceName)
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
		write := true
		for attempts := 0; write && attempts < 3; attempts++ {
			if attempts > 0 {
				fmt.Printf("attempt %d of 3\n", attempts+1)
			}
			bytesWritten, err := conWrite.WriteString(text)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("wrote", bytesWritten, "bytes")
			}
			flushErr := conWrite.Flush()
			if flushErr != nil {
				fmt.Println("connection timed out, redialing connection...")
				svc, err = ctx.Dial(serviceName) //dial the service using the given name
				if err != nil {
					log.Fatalf("error when re-dialing service name %s. %v", serviceName, err)
				}
				fmt.Println("reconnected.")
				conRead = bufio.NewReader(svc)
				conWrite = bufio.NewWriter(svc)
				continue
			} else {
				write = false
				fmt.Print("Sent    :", text)
				var read string
				for {
					read, err = conRead.ReadString('\n')
					if err != nil {
						fmt.Println(err)
						return // exit the program when it reads EOF
					}
					if strings.TrimSpace(read) != "" {
						break
					}
				}
				fmt.Println("Received:", read)
			}
		}
		if write {
			fmt.Println("failed to send after 3 attempts, dropping message")
		}
	}
}
