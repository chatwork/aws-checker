package main

import (
	"flag"
	"fmt"
)

func parseFlags(args []string) ([]Option, *int) {
	fs := flag.NewFlagSet("aws-checker", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "%s is a toolkit for checking availability of AWS services.\n", fs.Name())
		fmt.Fprintf(fs.Output(), "Run '%s -help' for usage.\n", fs.Name())
	}

	var code int

	if err := fs.Parse(args); err != nil {
		if err != flag.ErrHelp {
			code = 2
		}
	} else {
		switch fs.NArg() {
		case 0:
			return []Option{}, nil
		case 1:
			switch fs.Arg(0) {
			case "version":
				fmt.Fprintf(fs.Output(), "%s %s", fs.Name(), Version)
			default:
				fmt.Fprintf(fs.Output(), "unknown command %q for %q\n", fs.Arg(0), fs.Name())
				fmt.Fprintf(fs.Output(), "Run '%s -help' for usage.\n", fs.Name())
				code = 2
			}
		default:
			fmt.Fprintf(fs.Output(), "too many arguments\n")
			fmt.Fprintf(fs.Output(), "Run '%s -help' for usage.\n", fs.Name())
			code = 2
		}
	}

	return nil, &code
}
