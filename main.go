package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Random Run!")

	command := "c:/Users/richa/Documents/fba64_029743/fba64.exe"
	flags := []string{"vsav", "-w"}

	err := run(command, flags)
	if err != nil {
		log.Fatal(err)
	}
}

func run(command string, flags []string) error {
	cmd := exec.Command(command, flags...)
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return err
	}

	cmd.Wait()
	return nil
}
