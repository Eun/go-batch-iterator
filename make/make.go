// package main is just a simple app to run the tests and fuzz.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fuzztime := flag.Duration("fuzztime", time.Second*30, "")
	coverProfile := flag.String("coverprofile", "cover.cov", "")
	flag.Parse()

	if err := test(*coverProfile); err != nil {
		log.Fatal(err)
	}

	if err := fuzz(*fuzztime); err != nil {
		log.Fatal(err)
	}
}

func fuzz(fuzzTime time.Duration) error {
	var stdout bytes.Buffer
	cmd := exec.Command("go", "test", "-list", "^Fuzz", ".")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "Fuzz") {
			continue
		}
		//nolint: gosec // allow call with variable
		fuzzCmd := exec.Command("go",
			"test",
			"-v",
			"-fuzz",
			line,
			"-fuzztime",
			fuzzTime.String(),
			"-run",
			"^$",
			".")
		fuzzCmd.Stdout = os.Stdout
		fuzzCmd.Stderr = os.Stderr
		fmt.Println(strings.Join(fuzzCmd.Args, " "))
		if err := fuzzCmd.Run(); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func test(coverprofile string) error {
	cmd := exec.Command("go",
		"test",
		"-v",
		"-race",
		"-coverprofile",
		coverprofile,
		"-covermode",
		"atomic",
		"./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Println(strings.Join(cmd.Args, " "))
	return cmd.Run()
}
