package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/paul/gh-wiki/cmd"
)

func main() {
	if host := os.Getenv("GH_WIKI_ASKPASS_HOST"); host != "" {
		prompt := strings.ToLower(strings.Join(os.Args[1:], " "))
		if strings.Contains(prompt, "username") {
			fmt.Println("x-access-token")
			return
		}
		token, _ := auth.TokenForHost(host)
		fmt.Println(token)
		return
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
