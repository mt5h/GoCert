package main

import (
	"fmt"
	"go-cert/checker"
	"go-cert/cmd"
	"go-cert/tui"
)

func main() {
	cmd.LoadFlags()
	if cmd.UseTui == false {
		for _, endpoint := range cmd.Endpoint {
			res := checker.GetJsonCert(endpoint, cmd.Timeout)
			fmt.Println(res)
		}
		return

	}
	tui.Launch(cmd.Endpoint)

}
