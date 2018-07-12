package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

const bucketName string = "rand-run-bucket"

type Runnable struct {
	command string
	name    string
	flags   []string
}

type RunRecord struct {
	name    string
	seconds int
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
		if len(items) < 3 {
			log.Println("Skipping", l)
			continue
		}

		runnables = append(runnables, Runnable{
			name:    items[0],
			command: items[1],
			flags:   items[2:],
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

func secondsToTimeString(seconds int) string {
	h := seconds / 3600
	s := seconds - (h * 3600)
	m := s / 60
	s -= (m * 60)

	return fmt.Sprintf("%d hours, %d minutes, %d seconds", h, m, s)
}

func randomRun(runnables []Runnable, dbPath string) chan struct{} {
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
	runKey := r.name
	doneCh := make(chan struct{})
	go func() {
		db, err := bolt.Open(dbPath, 0600, nil)
		defer db.Close()
		if err != nil {
			log.Fatal(err)
		}
		timerDoneCh := make(chan struct{})
		err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return err
			}
			time.Sleep(time.Second * 1)
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
					fmt.Printf("\r%s", secondsToTimeString(totalSeconds))
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
	return doneCh
}

func listRunTime(dbPath string) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal("Db Open error", err)
	}
	runRecords := []RunRecord{}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			seconds, e := getIntFromByteSlice(v)
			if e != nil {
				log.Fatal(e)
			}
			runRecords = append(runRecords, RunRecord{name: string(k), seconds: seconds})
		}
		db.Close()
		sort.Slice(runRecords, func(i, j int) bool {
			return runRecords[i].seconds > runRecords[j].seconds
		})
		for i, r := range runRecords {
			fmt.Printf("%d. %s: %s\n", i+1, r.name, secondsToTimeString(r.seconds))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	csvPath := flag.String("csv", "test.csv", "Path to runnable csv, first the name, the next column is the command the rest are the flags")
	dbPath := flag.String("db", "rand-run.db", "Path to db")
	list := flag.Bool("list", false, "List the times for all programs in db")
	flag.Parse()

	if !*list {
		runnables, err := parseRunnableCSV(*csvPath)
		if err != nil {
			log.Fatal(err)
		}
		doneCh := randomRun(runnables, *dbPath)

		select {
		case <-doneCh:
			fmt.Println("\nGoodbye!")
		}
	} else {
		listRunTime(*dbPath)
	}

}
