package main

import (
	"github.com/spf13/pflag"
	"fmt"
	"os"
	"github.com/tensorflow/k8s/cmd/tf_operator/app/options"
	"github.com/tensorflow/k8s/cmd/tf_operator/app"
)

func main() {
	s := options.NewServerOption()
	s.AddFlags(pflag.CommandLine)

	pflag.Parse()

	if err := app.Run(s); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}
