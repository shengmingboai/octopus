package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/shengmingboai/octopus/internal/conf"
	"github.com/shengmingboai/octopus/internal/db"
	"github.com/shengmingboai/octopus/internal/op"
	"github.com/shengmingboai/octopus/internal/server"
	"github.com/shengmingboai/octopus/internal/task"
	"github.com/shengmingboai/octopus/internal/utils/shutdown"
	"github.com/spf13/cobra"
)

var cfgFile string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start " + conf.APP_NAME,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		conf.PrintBanner()
		if err := conf.Load(cfgFile); err != nil {
			return err
		}
		if level, err := log.ParseLevel(conf.AppConfig.Log.Level); err == nil {
			log.SetLevel(level)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		shutdown.Init(log.Default())
		defer shutdown.Shutdown()
		if err := db.InitDB(conf.AppConfig.Database.Type, conf.AppConfig.Database.Path, conf.IsDebug()); err != nil {
			return fmt.Errorf("database init: %w", err)
		}
		shutdown.Register(db.Close)

		if err := op.InitCache(); err != nil {
			return fmt.Errorf("cache init: %w", err)
		}
		shutdown.Register(op.SaveCache)

		if err := op.UserInit(); err != nil {
			return fmt.Errorf("user init: %w", err)
		}

		if err := task.Init(); err != nil {
			return fmt.Errorf("task init: %w", err)
		}
		if err := server.Start(); err != nil {
			return fmt.Errorf("server start: %w", err)
		}

		taskCtx, cancelTasks := context.WithCancel(context.Background())
		tasksDone := make(chan struct{})
		go func() {
			defer close(tasksDone)
			task.Run(taskCtx)
		}()
		shutdown.Register(func() error {
			cancelTasks()
			<-tasksDone
			return nil
		})
		shutdown.Register(server.Close)
		shutdown.Listen()
		return nil
	},
}

func init() {
	startCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./data/config.json)")
	rootCmd.AddCommand(startCmd)
}
