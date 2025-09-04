package service

import (
	"log"

	"github.com/go-co-op/gocron/v2"
)

func NewScheduler() gocron.Scheduler {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}
	return scheduler
}
