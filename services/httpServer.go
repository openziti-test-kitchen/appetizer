package services

import (
	"fmt"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func ServeHTTPOverZiti(serverIdentity *ziti.Config, svcName string) {
	listener := CreateZitiListener(serverIdentity, svcName)
	svr := &http.Server{}
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(hello))
	mux.Handle("/hello", http.HandlerFunc(hello))
	mux.Handle("/domath", http.HandlerFunc(mathHandler))

	svr.Handler = mux

	if err := svr.Serve(listener); err != nil {
		logrus.Fatal(err)
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	_, _ = fmt.Fprintf(w, "hello from %s\n", host)
}

func mathHandler(w http.ResponseWriter, r *http.Request) {
	input1, err := strconv.ParseFloat(r.URL.Query().Get("input1"), 64)
	if err != nil {
		http.Error(w, "Invalid input1", http.StatusBadRequest)
		return
	}

	input2, err := strconv.ParseFloat(r.URL.Query().Get("input2"), 64)
	if err != nil {
		http.Error(w, "Invalid input2", http.StatusBadRequest)
		return
	}

	var result float64

	switch r.URL.Query().Get("operator") {
	case "+":
		result = input1 + input2
	case "-":
		result = input1 - input2
	case "*":
		result = input1 * input2
	case "/":
		if input2 == 0 {
			http.Error(w, "Division by zero not allowed", http.StatusBadRequest)
			return
		}
		result = input1 / input2
	default:
		http.Error(w, "Invalid operator, Use +, -, *, or /", http.StatusBadRequest)
		return
	}

	_, _ = fmt.Fprintf(w, "Result: %.2f\n", result)
}

func CreateZitiListener(serverIdentity *ziti.Config, serviceName string) net.Listener {
	options := ziti.ListenOptions{
		ConnectTimeout: 5 * time.Minute,
	}
	ctx, err := ziti.NewContext(serverIdentity)

	if err != nil {
		logrus.Fatal(err)
	}

	listener, err := ctx.ListenWithOptions(serviceName, &options)
	if err != nil {
		fmt.Printf("Error binding service %+v\n", err)
		logrus.Fatal(err)
	}
	return listener
}
