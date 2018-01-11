package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tensorflow/k8s/cmd/tf_operator/app"
	"github.com/tensorflow/k8s/cmd/tf_operator/app/options"
)

func main() {
	s := options.NewServerOption()
	s.AddFlags(flag.CommandLine)

	flag.Parse()

	if err := app.Run(s); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}
