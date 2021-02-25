package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

func main() {
	fDebug := flag.Bool("debug", false, "enable debug logging")
	fOutDir := flag.String("outdir", "out", "the directory in which the output files should be stored")

	flag.Parse()

	var loggerOut io.Writer

	if !*fDebug {
		loggerOut = ioutil.Discard
	} else {
		loggerOut = os.Stdout
	}

	log.SetOutput(loggerOut)

	log.Printf("Storing the ouputs in %s", *fOutDir)

	var wg sync.WaitGroup
	wg.Add(flag.NArg())

	for _, input := range flag.Args() {
		go func(i string) {
			process(
				i,
				*fOutDir,
				log.New(loggerOut, fmt.Sprintf("%s | ", i), 0),
			)

			wg.Done()
		}(input)
	}

	log.Print("Waiting for the goroutines")

	wg.Wait()
}

func process(in, outdir string, logger *log.Logger) {
	sim, err := Parse(in)
	if err != nil {
		logger.Fatalf("Failed to parse %s: %v", in, err)
	}

	logger.Printf("%s: %+v", in, sim)

	_ = sim
}

func Parse(in string) (*Simulation, error) {
	s := Simulation{}

	fd, err := os.Open(in)
	if err != nil {
		return nil, err
	}

	var (
		streets int
		cars    int
	)

	if _, err := fmt.Fscanf(fd, "%d %d %d %d %d", &s.Duration, &s.Intersections, &streets, &cars, &s.Bonus); err != nil {
		return nil, fmt.Errorf("could not parse header: %v", err)
	}

	s.CarPaths = make([][]string, 0, cars)
	s.Streets = make(map[string]*Street, streets)

	for i := 0; i < streets; i++ {
		st := Street{}

		if _, err := fmt.Fscanf(fd, "%d %d %s %d", &st.Begin, &st.End, &st.Name, &st.Time); err != nil {
			return nil, fmt.Errorf("could not parse street %d: %v", i, err)
		}

		s.Streets[st.Name] = &st
	}

	for i := 0; i < cars; i++ {
		var (
			name     string
			nstreets int
		)

		if _, err := fmt.Fscanf(fd, "%d", &nstreets); err != nil {
			return nil, fmt.Errorf("could not parse the number of streets: %v", err)
		}

		path := make([]string, 0, nstreets)

		for j := 0; j < nstreets; j++ {
			if _, err := fmt.Fscanf(fd, "%s", &name); err != nil {
				return nil, fmt.Errorf("could not parse street %d: %v", j, err)
			}

			path = append(path, name)
		}

		s.CarPaths = append(s.CarPaths, path)
	}

	return &s, nil
}

type Simulation struct {
	Duration      int
	Intersections int
	Streets       map[string]*Street
	CarPaths      [][]string
	Bonus         int
}

type Street struct {
	Begin int
	End   int
	Name  string
	Time  int
}
