package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
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

type (
	Output           = map[int]Intersection
	Intersection     map[string]*IntersectionItem
	IntersectionItem struct {
		GreenTime int
		Weight    float64 // number of cars going through the intersection
	}
)

func (i Intersection) CalculateGreenTimes(maxDuration int) {
	lowest := math.MaxFloat64

	for _, item := range i {
		if item.Weight < lowest {
			lowest = item.Weight
		}
	}

	lowest *= 3

	for _, item := range i {
		//item.GreenTime = int(
		//	math.Min(
		//		math.Max(math.Floor(item.Weight/lowest), 1),
		//		float64(maxDuration),
		//	),
		//)

		item.GreenTime = 2
	}
}

func process(in, outdir string, logger *log.Logger) {
	sim, err := Parse(in)
	if err != nil {
		logger.Fatalf("Failed to parse %s: %v", in, err)
	}

	// Remove cars that won't make it
	//sim.RemoveCarPercent(80)

	usedStreets := sim.UsedStreets()

	output := make(Output)

	for streetName, street := range sim.Streets {
		weight := usedStreets[streetName]

		if weight == 0 {
			//logger.Printf("%s: discarding street %s", in, streetName)
			continue
		}

		if output[street.End] == nil {
			output[street.End] = make(Intersection)
		}

		output[street.End][streetName] = &IntersectionItem{Weight: weight}
	}

	for _, intersection := range output {
		intersection.CalculateGreenTimes(sim.Duration)
	}

	outFile := filepath.Join(outdir, filepath.Base(in))

	if err := WriteOutput(outFile, output); err != nil {
		logger.Fatalf("Could not write %s: %v", outFile, err)
	}
}

func Parse(in string) (*Simulation, error) {
	s := Simulation{}

	fd, err := os.Open(in)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	var (
		streets int
		cars    int
	)

	if _, err := fmt.Fscanf(fd, "%d %d %d %d %d", &s.Duration, &s.Intersections, &streets, &cars, &s.Bonus); err != nil {
		return nil, fmt.Errorf("could not parse header: %v", err)
	}

	s.CarPaths = make([]CarPath, 0, cars)
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

func WriteOutput(filename string, output Output) error {
	fd, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fd.Close()

	fmt.Fprintf(fd, "%d\n", len(output))

	for intersectionID, intersection := range output {
		// Write the intersection ID
		fmt.Fprintf(fd, "%d\n%d\n", intersectionID, len(intersection))

		for streetName, item := range intersection {
			fmt.Fprintf(fd, "%s %d\n", streetName, item.GreenTime)
		}
	}

	return nil
}

type Streets map[string]*Street

type Simulation struct {
	Duration      int
	Intersections int
	Streets       Streets
	CarPaths      []CarPath
	Bonus         int
}

func (s *Simulation) RemoveCarPercent(pc int) {
	// Sort the cars by profit
	sort.Slice(s.CarPaths, func(i, j int) bool {
		return s.Duration-s.CarPaths[i].Deadline(s.Streets) < s.Duration-s.CarPaths[j].Deadline(s.Streets)
	})

	elemsToRemove := (float64(pc) / 100) * float64(len(s.CarPaths))

	log.Printf("Removing %f elems from %d cars", elemsToRemove, len(s.CarPaths))

	s.CarPaths = s.CarPaths[int(elemsToRemove):]
}

type CarPath []string

func (s *Simulation) UsedStreets() map[string]float64 {
	used := make(map[string]float64)

	for _, carPath := range s.CarPaths {
		for _, streetName := range carPath {
			used[streetName]++
		}
	}

	return used
}

func (s *Simulation) UsedStreetsDividedByTime() map[string]float64 {
	used := make(map[string]float64)

	for _, carPath := range s.CarPaths {
		for _, streetName := range carPath {
			used[streetName]++
		}
	}

	for streetName, ncars := range used {
		used[streetName] = ncars / float64(s.Streets[streetName].Time)
	}

	return used
}

type Street struct {
	Begin int
	End   int
	Name  string
	Time  int
}

func (c CarPath) Deadline(streets Streets) int {
	deadline := 0

	for _, s := range c[1:] { // skip the first street
		deadline += streets[s].Time
	}

	return deadline
}
