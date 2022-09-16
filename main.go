//github.com/bigwhite/experiments/http-keep-alive/client-keepalive-on/client.go
package main

import (
	"context"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Cfg struct {
	SleepDuration          time.Duration `yaml:"sleep-duration" env:"SLEEP_DURATION" env-default:"1s"`
	SleepBeforeTermination time.Duration `yaml:"sleep-before-termintation" env:"SLEEP_BEFORE_TERMINATION" env-default:"15s"`
	Url                    string        `yaml:"url" env:"URL" env-default:"http://ifconfig.me"`
	Method                 string        `yaml:"method" env:"METHOD" env-default:"GET"`
}

func makeReq(ctx context.Context, c *http.Client, req *http.Request, sleep time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := c.Do(req)
			if err != nil {
				fmt.Println("http get error:", err)
				return
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("read body error:", err)
				return
			}
			fmt.Printf("time: %v response body: %s\n", time.Now(), string(b))
			time.Sleep(sleep)
		}
	}
}

//https://github.com/bigwhite/experiments/blob/master/http-keep-alive/client-keepalive-on/client-with-delay.go
func main() {
	var cfg Cfg

	err := cleanenv.ReadConfig("config.yaml", &cfg)
	if err != nil {
		log.Println("Can't load config... skipping")
	}
	err = cleanenv.ReadEnv(&cfg)
	if err != nil {
		log.Println("Can't load env")
	}

	t := &http.Transport{
		DialContext: (&net.Dialer{
			KeepAlive: 5 * time.Second,
		}).DialContext,
		IdleConnTimeout: 5 * time.Second,
	}
	c := &http.Client{
		Transport: t,
		//Timeout:   time.Second * 5,
	}
	req, err := http.NewRequest(cfg.Method, cfg.Url, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Request: %#v\n", *req)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		s := <-sigChan
		close(sigChan)
		log.Println("Catch signal: ", s)
		log.Printf("Sleep for %v \n", cfg.SleepBeforeTermination)
		time.Sleep(cfg.SleepBeforeTermination)
		cancel()
	}()
	makeReq(ctx, c, req, cfg.SleepDuration)
	c.CloseIdleConnections()
	fmt.Println("Well done")
}
