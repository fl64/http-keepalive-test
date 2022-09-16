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
	SleepDuration                  time.Duration `yaml:"sleep-duration" env:"SLEEP_DURATION" env-default:"1s"`
	SleepBeforeTermination         time.Duration `yaml:"sleep-before-termintation" env:"SLEEP_BEFORE_TERMINATION" env-default:"15s"`
	Url                            string        `yaml:"url" env:"URL" env-default:"http://ifconfig.me"`
	Method                         string        `yaml:"method" env:"METHOD" env-default:"GET"`
	DialerTimeout                  time.Duration `yaml:"dialer-timeout" env:"DIALER_TIMEOUT" env-default:"5s"`
	DialerKeepAlive                time.Duration `yaml:"dialer-keepalive" env:"DIALER_KEEPALIVE" env-default:"5s"`
	TransportIdleConnectionTimeout time.Duration `yaml:"idle-connection-timeout" env:"IDLE_CONNECTION_TIMEOUT" env-default:"5s"`
	ClientTimeout                  time.Duration `yaml:"client-timeout" env:"CLIENT_TIMEOUT" env-default:"5s"`
}

func makeReq(ctx context.Context, c *http.Client, req *http.Request, sleep time.Duration) error {
	var counter int
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			resp, err := c.Do(req)
			if err != nil {
				fmt.Println("http get error:", err)
				return err
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("read body error:", err)
				return err
			}
			fmt.Printf("counter: %d time: %v response body: %s\n", counter, time.Now(), string(b))
			time.Sleep(sleep)
			counter++
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
			Timeout:   cfg.DialerTimeout,
			KeepAlive: cfg.DialerKeepAlive,
		}).DialContext,
		IdleConnTimeout: cfg.TransportIdleConnectionTimeout,
	}
	c := &http.Client{
		Transport: t,
		Timeout:   cfg.ClientTimeout,
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
	err = makeReq(ctx, c, req, cfg.SleepDuration)
	if err != nil {
		log.Panic(err)
	}
	c.CloseIdleConnections()
	fmt.Println("Well done")
}
