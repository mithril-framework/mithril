package main

import (
	"fmt"
	"os"
	"strings"

	"mithril-rev/pkg/utils"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: hash|sha256|sha512|encode|decode <value>")
		os.Exit(1)
	}

	var s string
	if len(os.Args) >= 3 {
		s = strings.Join(os.Args[2:], " ")
	} else {
		s = os.Getenv("S")
	}
	if s == "" {
		fmt.Fprintln(os.Stderr, "Usage: make hash <password>   or   make encode <string>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "hash":
		out, err := utils.HashPassword(s)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(out)
	case "sha256":
		fmt.Println(utils.SHA256(s))
	case "sha512":
		fmt.Println(utils.SHA512(s))
	case "encode":
		fmt.Println(utils.EncodeBase64(s))
	case "decode":
		out, err := utils.DecodeBase64(s)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(out)
	default:
		fmt.Fprintln(os.Stderr, "Usage: hash|sha256|sha512|encode|decode")
		os.Exit(1)
	}
}
