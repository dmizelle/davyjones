package main

import (
	"os"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var log *zap.SugaredLogger

var nodes []string
var cfg Config

type nodeLabel struct {
	Label string
	Value string
}

type daemonSet struct {
	Namespace string
	Name      string
}

type Config struct {
	NodeLabels []nodeLabel
	DaemonSets []daemonSet
	Evict      bool
}

func main() {

	flag.String("resync", "5", "refresh rate for informer")
	flag.String("config", "/davyjones.yaml", "config file path")
	flag.Bool("debug", false, "print extra logging information")
	flag.Parse()

	viper.BindPFlags(flag.CommandLine)

	debug := viper.GetBool("debug")

	var zapper *zap.Logger
	var err error

	if debug {
		zapper, err = zap.NewDevelopment()
	} else {
		zapper, err = zap.NewProduction()
	}
	if err != nil {
		panic(err)
	}
	log = zapper.Sugar()

	viper.SetConfigType("yaml")
	viper.SetConfigFile(viper.GetString("config"))

	err = viper.ReadInConfig()
	if err != nil {
		log.Fatalw("unable to read configuration file",
			"error", err.Error(),
		)
	}
	log.Debugw("read config from file",
		"path", viper.GetString("config"),
	)
	viper.Unmarshal(&cfg)
	log.Debugw("config generated",
		"config", cfg,
	)

	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalw("unable to build kubeconfig",
			"error", err.Error(),
		)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalw("unable to create kubernetes clientset",
			"error", err.Error(),
		)
	}

	watcher := NewWatcher(clientSet, &cfg)

	watcher.Run()
}
