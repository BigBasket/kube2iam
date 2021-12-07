package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/jtblin/kube2iam/iam"
)

func main() {
	client, _ := iam.NewClient("", true, "")

	iteration := 0
	for {

		wg := new(sync.WaitGroup)
		wg.Add(150)

		for index := 0; index < 150; index++ {
			go func() {
				iteration = iteration + rand.Int()

				start := time.Now()
				client.AssumeRole("arn:aws:iam::274334742953:role/dev-qa-k8s-msvcs-sentinels", "",
					strconv.Itoa(iteration), 900*time.Second)
				end := time.Since(start).Seconds()

				fmt.Printf("total time taken %v", end)
				fmt.Println()
				wg.Done()
			}()
		}

		wg.Wait()

		time.Sleep(30 * time.Second)
	}
}
