package nsqshell

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mreiferson/go-options"
	"xchg.ai/sse/nsq/nsqd"
	"xchg.ai/sse/nsqshell/internal/version"
)

type program struct {
	once sync.Once
	nsqd *nsqd.NSQD
}

func (p *program) Start(configFile string) error {
	opts := nsqd.NewOptions()

	flagSet := NsqdFlagSet(opts)
	flagSet.Parse(os.Args[1:])

	rand.Seed(time.Now().UTC().UnixNano())

	if flagSet.Lookup("version").Value.(flag.Getter).Get().(bool) {
		fmt.Println(version.String("nsqd"))
		os.Exit(0)
	}

	var cfg config
	if configFile != "" {
		_, err := toml.DecodeFile(configFile, &cfg)
		if err != nil {
			logFatal("failed to load config file %s - %s", configFile, err)
		}
	}
	cfg.Validate()

	options.Resolve(opts, flagSet, cfg)
	nsqd, err := nsqd.New(opts)
	if err != nil {
		logFatal("failed to instantiate nsqd - %s", err)
	}
	p.nsqd = nsqd

	err = p.nsqd.LoadMetadata()
	if err != nil {
		logFatal("failed to load metadata - %s", err)
	}
	err = p.nsqd.PersistMetadata()
	if err != nil {
		logFatal("failed to persist metadata - %s", err)
	}

	go func() {
		err := p.nsqd.Main()
		if err != nil {
			p.Stop()
			os.Exit(1)
		}
	}()

	return nil
}

func (p *program) Stop() error {
	p.once.Do(func() {
		p.nsqd.Exit()
	})
	return nil
}

func StartNsqdInternal(done <-chan interface{}, closed chan<- interface{}, configFile string) error {
	prg := &program{}
	if err := prg.Start(configFile); err != nil {
		fmt.Printf("NSQD Start Error: %v\n", err.Error())
		return err
	}

	for {
		select {
		case <-done:
			if err := prg.Stop(); err != nil {
				fmt.Println("NSQD Stop Error: %v\n", err.Error())
				return err
			}
			closed <- struct{}{}
		}
	}

	return nil
}
