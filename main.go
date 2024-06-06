package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/jd-opensource/joylive-injector/pkg/admission"
	_ "github.com/jd-opensource/joylive-injector/pkg/mutation"

	"github.com/jd-opensource/joylive-injector/pkg/config"

	"github.com/jd-opensource/joylive-injector/pkg/route"
	"github.com/spf13/cobra"
)

var (
	buildDate   string
	buildCommit string

	versionTpl = `
Name: joylive-injector
Arch: %s
BuildDate: %s
BuildCommit: %s
`
)

var rootCmd = &cobra.Command{
	Use:     "joylive-injector",
	Short:   "JoyLive dynamic admission control tool",
	Version: buildCommit,
	Run: func(cmd *cobra.Command, args []string) {
		log.InitLog()
		admission.Setup()
		route.Setup()

		srv := &http.Server{
			Handler: route.Router(),
			Addr:    config.Addr,
		}

		sigs := make(chan os.Signal)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			var shutdownOnce sync.Once
			for range sigs {
				log.Warn("Receiving the termination signal, graceful shutdown...")
				shutdownOnce.Do(func() {
					err := srv.Shutdown(context.Background())
					if err != nil {
						log.Error("Error:", zap.Error(err))
					}
				})
			}
		}()

		if config.Cert != "" && config.Key != "" {
			log.Infof("Listen TLS Server at %s", config.Addr)
			err := srv.ListenAndServeTLS(config.Cert, config.Key)
			if err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					log.Info("server shutdown success.")
				} else {
					log.Error("Error:", zap.Error(err))
				}
			}
		} else {
			log.Infof("Listen HTTP Server at %s", config.Addr)
			err := srv.ListenAndServe()
			if err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					log.Info("server shutdown success.")
				} else {
					log.Error("Error:", zap.Error(err))
				}
			}
		}
	},
}

func init() {
	// version template
	rootCmd.SetVersionTemplate(fmt.Sprintf(versionTpl, runtime.GOOS+"/"+runtime.GOARCH, buildDate, buildCommit))

	// webhook
	rootCmd.PersistentFlags().StringVar(&config.ConfigPath, "config", config.DefaultConfigPath, "Config file path")
	//rootCmd.PersistentFlags().StringVar(&config.ConfigMountPath, "config-mount-path", "/joylive/config", "Config file mount path")
	rootCmd.PersistentFlags().StringVar(&config.ConfigMountSubPath, "config-mount-sub-path", "config", "Config file mount sub path")
	rootCmd.PersistentFlags().StringVarP(&config.Addr, "listen", "l", ":443", "Admission Controller listen address")
	rootCmd.PersistentFlags().StringVar(&config.Cert, "cert", "", "Admission Controller TLS cert")
	rootCmd.PersistentFlags().StringVar(&config.Key, "key", "", "Admission Controller TLS cert key")
	rootCmd.PersistentFlags().StringVar(&config.MatchLabel, "match-label", "x-live", "Match label")

	// admission config
	rootCmd.PersistentFlags().StringVar(&config.InitContainerName, "init-container-name", "joylive-init-container", "Init container name")
	//rootCmd.PersistentFlags().StringVar(&config.InitContainerImage, "init-container-image", "busybox", "Init container image")
	//rootCmd.PersistentFlags().StringVar(&config.InitEmptyDirMountPath, "init-empty-dir-mount-path", "/agent", "Init empty dir mount path")
	//rootCmd.PersistentFlags().StringVar(&config.InitContainerCmd, "init-container-cmd", "/bin/sh", "Init container cmd")
	//rootCmd.PersistentFlags().StringVar(&config.InitContainerArgs, "init-container-args", fmt.Sprintf("-c, cp -r /joylive/* %s && chmod -R 777 %s",
	//	config.InitEmptyDirMountPath, config.InitEmptyDirMountPath), "Init container cmd")
	//rootCmd.PersistentFlags().StringVar(&config.InitContainerEnvKey, "init-container-env-key", "JAVA_TOOL_OPTIONS", "Init container env key")
	//rootCmd.PersistentFlags().StringVar(&config.InitContainerEnvValue, "init-container-env-value", "-javaagent:/joylive/live.jar", "Init container env value")
	//rootCmd.PersistentFlags().StringVar(&config.EmptyDirMountPath, "empty-dir-mount-path", "/joylive", "Empty dir mount path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("Start error.", zap.Error(err))
	}
}
