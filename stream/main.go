package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

func inputStream(spoolfile string, stdline []byte) {
	f, err := os.OpenFile(spoolfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	if _, err := f.Write(stdline); err != nil {
		fmt.Println(err)
	}
}

func main() {

	if len(os.Args) == 1 {
		fmt.Printf(" Script file name required as command-line arg")
	}

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	// Current User
	fmt.Println("Hi " + user.Name + " (id: " + user.Uid + ")")
	fmt.Println("Username: " + user.Username)
	fmt.Println("Home Dir: " + user.HomeDir)

	// Get "Real" User under sudo.
	// More Info: https://stackoverflow.com/q/29733575/402585
	fmt.Println("Real User: " + os.Getenv("SUDO_USER"))
	daemon := exec.Command("./daemon/daemon", fmt.Sprintf("%s", os.Args[1]), "&")
	e := daemon.Start()
	if e != nil {
		fmt.Println(e.Error())
	}
}
