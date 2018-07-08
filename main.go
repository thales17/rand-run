package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

type Runnable struct {
	command string
	flags   []string
}

func run(r Runnable) error {
	fmt.Printf("Randomly Running: %s %s", r.command, strings.Join(r.flags, " "))
	cmd := exec.Command(r.command, r.flags...)
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return err
	}

	cmd.Wait()
	return nil
}

func parseRunnableCSV(csvPath string) ([]Runnable, error) {
	content, err := ioutil.ReadFile(csvPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	runnables := []Runnable{}
	for _, l := range lines {
		items := strings.Split(l, ",")
		if len(items) < 2 {
			continue
		}
		runnables = append(runnables, Runnable{
			command: items[0],
			flags:   items[1:],
		})
	}

	return runnables, nil
}

func getByteSliceFromInt(num int) []byte {
	return []byte(strconv.Itoa(num))
}

func getIntFromByteSlice(bs []byte) (int, error) {
	return strconv.Atoi(string(bs))
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	csvPath := flag.String("csv", "test.csv", "Path runnable csv, first column is the command the rest are the flags")
	flag.Parse()

	runnables, err := parseRunnableCSV(*csvPath)
	if err != nil {
		log.Fatal(err)
	}
	randIndex := rand.Intn(len(runnables))
	runDoneCh := make(chan struct{})
	go func() {
		err := run(runnables[randIndex])
		close(runDoneCh)
		if err != nil {
			log.Fatal(err)
		}
	}()
	r := runnables[randIndex]
	runKey := fmt.Sprintf("%s %s", r.command, strings.Join(r.flags, " "))
	doneCh := make(chan struct{})
	go func() {
		db, err := bolt.Open("rand-run.db", 0600, nil)
		defer db.Close()
		if err != nil {
			log.Fatal(err)
		}
		timerDoneCh := make(chan struct{})
		err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte("rand-run-bucket"))
			if err != nil {
				return err
			}
			//			fmt.Println("\n*****\nBucket:", b)
			time.Sleep(time.Second * 1)
			//		fmt.Println("\n******\nRunKey:", runKey)
			totalSeconds, err := getIntFromByteSlice(b.Get([]byte(runKey)))
			if err != nil {
				log.Println(err)
				totalSeconds = 0
			}
			fmt.Println("\n***************************")
			for {
				select {
				case <-runDoneCh:
					close(timerDoneCh)
					return nil
				default:
					hours := totalSeconds / 3600
					seconds := totalSeconds - (hours * 3600)
					minutes := seconds / 60
					seconds -= (minutes * 60)

					fmt.Printf("\r%d hours, %d minutes, %d seconds", hours, minutes, seconds)
					time.Sleep(time.Second)
					err := b.Put([]byte(runKey), getByteSliceFromInt(totalSeconds))
					if err != nil {
						close(timerDoneCh)
						return err
					}
					totalSeconds += 1
				}
			}
			return nil
		})

		if err != nil {
			log.Fatal(err)
		}

		select {
		case <-timerDoneCh:
			close(doneCh)
		}
	}()

	select {
	case <-doneCh:
		fmt.Println("\nGoodbye!")
	}
}
