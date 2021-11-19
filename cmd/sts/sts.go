package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/jtblin/kube2iam/iam"
)

func main() {
	client, _ := iam.NewClient("", true, "")

	iteration := 0
	for {

		iteration = iteration + rand.Int()

		start := time.Now()
		client.AssumeRole("arn:aws:iam::274334742953:role/dev-qa-k8s-msvcs-sentinels", "",
			strconv.Itoa(iteration), 900*time.Second)
		end := time.Since(start).Seconds()

		fmt.Printf("total time taken %v", end)
		fmt.Println()

		time.Sleep(1 * time.Second)
	}
}
