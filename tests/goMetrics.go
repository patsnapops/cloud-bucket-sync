package main

import (
	"fmt"
	"time"

	"github.com/rcrowley/go-metrics"
	influxdb "github.com/vrischmann/go-metrics-influxdb"
)

func main() {
	c := metrics.NewCounter()
	metrics.Register("money", c)
	c.Inc(17)

	// Threadsafe registration
	t := metrics.GetOrRegisterTimer("db.get.latency", nil)
	t.Time(func() {})
	t.Update(1)

	fmt.Println(c.Count())
	fmt.Println(t.Min())

	go influxdb.InfluxDB(metrics.DefaultRegistry,
		123*time.Millisecond,
		"127.0.0.1:8086",
		"database-name",
		"measurement-name",
		"username",
		"password",
		false,
	)

}
