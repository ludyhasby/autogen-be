package helperconverter

import (
	"errors"
	"time"

	"github.com/robfig/cron/v3"
)

func CronExprToIntervalMinutes(expr string, defaultInMinutes int) (minutes int, err error) {
	minutes = defaultInMinutes

	sched, err := cron.ParseStandard(expr)
	if err != nil {
		return minutes, err
	}

	now := time.Now()
	next := sched.Next(now)
	next2 := sched.Next(next)

	interval := next2.Sub(next)
	minutes = int(interval.Minutes())

	//check interval valid or interval lebih 7 hari
	if interval <= 0 || interval > 7*24*time.Hour {
		return minutes, errors.New("Invalid Cron Expr")
	}

	return minutes, nil
}
