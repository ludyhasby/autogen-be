package helpergenerator

import (
	"fmt"
	"time"
)

func GetLatency(timeIn time.Time) string {
	ms := time.Now().Sub(timeIn).Seconds()
	return fmt.Sprintf("%.2f ms", ms)
}
