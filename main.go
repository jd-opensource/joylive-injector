package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/jd-opensource/joylive-injector/pkg/log"
	"go.uber.org/zap"

	"github.com/jd-opensource/joylive-injector/pkg/admission"
	_ "github.com/jd-opensource/joylive-injector/pkg/apm"
	_ "github.com/jd-opensource/joylive-injector/pkg/mutation"
	_ "github.com/jd-opensource/joylive-injector/pkg/watcher"

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
	rootCmd.PersistentFlags().StringVarP(&config.Addr, "listen", "l", ":443", "Admission Controller listen address")
	rootCmd.PersistentFlags().StringVar(&config.Cert, "cert", "", "Admission Controller TLS cert")
	rootCmd.PersistentFlags().StringVar(&config.Key, "key", "", "Admission Controller TLS cert key")
	rootCmd.PersistentFlags().StringVar(&config.MatchLabels, "match-label", config.MatchLabels, "Match label")
	rootCmd.PersistentFlags().StringVar(&config.ControlPlaneUrl, "control-plane-url", config.ControlPlaneUrl, "Control Plane URL")
	rootCmd.PersistentFlags().StringVar(&config.ClusterId, "cluster-id", config.ClusterId, "Cluster ID")
	//rootCmd.PersistentFlags().StringVar(&config.KubeConfig, "kubeconfig", config.KubeConfig, "Path to the kubeconfig file")

	// admission config
	rootCmd.PersistentFlags().StringVar(&config.InitContainerName, "init-container-name", "joylive-init-container", "Init container name")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("Start error.", zap.Error(err))
	}
}
