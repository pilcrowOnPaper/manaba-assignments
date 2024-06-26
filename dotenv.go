package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func loadEnv() {
	envFile, err := os.Open(".env")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		pair := strings.Split(scanner.Text(), "=")
		if pair[0] == "USERNAME" {
			USERNAME = strings.Trim(pair[1], "\"")
		}
		if pair[0] == "PASSWORD" {
			PASSWORD = strings.Trim(pair[1], "\"")
		}
	}
}
