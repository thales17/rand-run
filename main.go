package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

type runnable struct {
	command string
	flags   []string
}

func run(r runnable) error {
	cmd := exec.Command(r.command, r.flags...)
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return err
	}

	cmd.Wait()
	return nil
}

func parseRunnableCSV(csvPath string) ([]runnable, error) {
	content, err := ioutil.ReadFile(csvPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	runnables := []runnable{}
	for _, l := range lines {
		items := strings.Split(l, ",")
		if len(items) < 2 {
			continue
		}
		runnables = append(runnables, runnable{
			command: items[0],
			flags:   items[1:],
		})
	}
	// TODO: return created runnable
	/*	games := []string{"vsav", "invaders"}
		runnables := []runnable{}
		for _, g := range games {
			r := runnable{
				command: "c:/Users/richa/Documents/fba64_029743/fba64.exe",
				flags:   []string{g, "-w"},
			}
			runnables = append(runnables, r)
		}*/

	return runnables, nil
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	fmt.Println("Random Run!")

	runnables, err := parseRunnableCSV("test.csv")
	if err != nil {
		log.Fatal(err)
	}
	randIndex := rand.Intn(len(runnables))
	doneCh := make(chan struct{})
	go func() {
		err := run(runnables[randIndex])
		close(doneCh)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		totalSeconds := 0
		for {
			hours := totalSeconds / 3600
			seconds := totalSeconds - (hours * 3600)
			minutes := seconds / 60
			seconds -= (minutes * 60)

			fmt.Printf("\r%d hours, %d minutes, %d seconds", hours, minutes, seconds)
			time.Sleep(time.Second)
			totalSeconds += 1
		}
	}()

	select {
	case <-doneCh:
		fmt.Println("\nGoodbye!")
	}
}
