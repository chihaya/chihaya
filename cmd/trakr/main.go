package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/jzelinskie/trakr"
)

func main() {
	var configFilePath string
	var cpuProfilePath string

	var rootCmd = &cobra.Command{
		Use:   "trakr",
		Short: "BitTorrent Tracker",
		Long:  "A customizible, multi-protocol BitTorrent Tracker",
		Run: func(cmd *cobra.Command, args []string) {
			if err := func() error {
				if cpuProfilePath != "" {
					log.Println("enabled CPU profiling to " + cpuProfilePath)
					f, err := os.Create(cpuProfilePath)
					if err != nil {
						return err
					}
					pprof.StartCPUProfile(f)
					defer pprof.StopCPUProfile()
				}

				mt, err := trakr.MultiTrackerFromFile(configFilePath)
				if err != nil {
					return errors.New("failed to read config: " + err.Error())
				}

				go func() {
					shutdown := make(chan os.Signal)
					signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
					<-shutdown
					mt.Stop()
				}()

				if err := mt.ListenAndServe(); err != nil {
					return errors.New("failed to cleanly shutdown: " + err.Error())
				}

				return nil
			}(); err != nil {
				log.Fatal(err)
			}
		},
	}

	rootCmd.Flags().StringVar(&configFilePath, "config", "/etc/trakr.yaml", "location of configuration file (defaults to /etc/trakr.yaml)")
	rootCmd.Flags().StringVarP(&cpuProfilePath, "cpuprofile", "", "", "location to save a CPU profile")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
