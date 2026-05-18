package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/openlibrecommunity/olcrtc/internal/subscription"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "deploy/olcrtc/desired.yaml", "desired provisioning YAML")
	statePath := flag.String("state", "deploy/olcrtc/state.yaml", "generated state YAML")
	check := flag.Bool("check", false, "fail if state is not up to date")
	printURLs := flag.Bool("print-urls", true, "print generated subscription URLs")
	flag.Parse()

	desired, err := subscription.LoadDesired(*configPath)
	if err != nil {
		return fmt.Errorf("load desired config: %w", err)
	}
	previous, err := subscription.LoadStateIfExists(*statePath)
	if err != nil {
		return fmt.Errorf("load previous state: %w", err)
	}
	next, err := subscription.Generate(desired, previous, subscription.GenerateOptions{})
	if err != nil {
		return fmt.Errorf("generate state: %w", err)
	}

	if *check {
		ok, err := sameYAML(previous, next)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s is not up to date; run olcrtc-provision", *statePath)
		}
		return nil
	}

	if err := subscription.SaveState(*statePath, next); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	if *printURLs {
		for _, user := range next.Users {
			fmt.Printf("%s %s\n", user.ID, user.SubscriptionURL)
		}
	}
	return nil
}

func sameYAML(a, b subscription.State) (bool, error) {
	ay, err := yaml.Marshal(a)
	if err != nil {
		return false, fmt.Errorf("marshal previous state: %w", err)
	}
	by, err := yaml.Marshal(b)
	if err != nil {
		return false, fmt.Errorf("marshal generated state: %w", err)
	}
	return bytes.Equal(ay, by), nil
}
