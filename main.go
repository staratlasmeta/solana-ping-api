package main

import (
	"flag"
	"log"
	"time"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var config Config

// Cluster enum
type Cluster string

var database *gorm.DB

const useGCloudDB = true

type ClustersToRun string

// Cluster enum
const (
	MainnetBeta Cluster = "MainnetBeta"
	Testnet             = "Testnet"
	Devnet              = "Devnet"
	Atlasnet            = "Atlasnet"
)

var influxdb *InfluxdbClient
var userInputClusterMode string
var atlasnetFailover RPCFailover

const (
	RunMainnetBeta ClustersToRun = "mainnet"
	RunTestnet                   = "testnet"
	RunDevnet                    = "devnet"
	RunAtlasnet                  = "atlasnet"
	RunAllClusters               = "all"
)

func init() {
	config = loadConfig()
	log.Println(" *** Config Start *** ")
	log.Println("--- //// Database Config --- ")
	log.Println(config.Database)
	log.Println("--- //// Influxdb Config --- ")
	log.Println(config.InfluxdbConfig)
	log.Println("--- //// Retension --- ")
	log.Println(config.Retension)
	log.Println("--- //// ClusterCLIConfig--- ")
	log.Println("ClusterCLIConfig Atlasnet", config.ClusterCLIConfig.ConfigAtlasnet)
	log.Println("--- Atlasnet Ping  --- ")
	log.Println("Atlasnet.ClusterPing.APIServer", config.Atlasnet.ClusterPing.APIServer)
	log.Println("Atlasnet.ClusterPing.PingServiceEnabled", config.Atlasnet.ClusterPing.PingServiceEnabled)
	log.Println("Atlasnet.ClusterPing.AlternativeEnpoint.HostList", config.Atlasnet.ClusterPing.AlternativeEnpoint.HostList)
	log.Println("Atlasnet.ClusterPing.PingConfig", config.Atlasnet.ClusterPing.PingConfig)
	log.Println("Atlasnet.ClusterPing.Report", config.Atlasnet.ClusterPing.Report)

	log.Println(" *** Config End *** ")

	ResponseErrIdentifierInit()
	StatisticErrExpectionInit()
	AlertErrExpectionInit()
	ReportErrExpectionInit()
	PingTakeTimeErrExpectionInit()

	if config.Database.UseGoogleCloud {
		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			DriverName: "cloudsqlpostgres",
			DSN:        config.DBConn,
		}))
		if err != nil {
			log.Panic(err)
		}
		database = gormDB
	} else {
		gormDB, err := gorm.Open(postgres.Open(config.DBConn), &gorm.Config{})
		if err != nil {
			log.Panic(err)
		}
		database = gormDB
	}
	err := database.AutoMigrate(&PingResult{})
	if err != nil {
		log.Printf("Failed to auto migrate: %v", err)
	}
	log.Println("database connected")
	if config.InfluxdbConfig.Enabled {
		influxdb = NewInfluxdbClient(config.InfluxdbConfig)
	}
	/// ---- Start RPC Failover ---
	log.Println("RPC Endpoint Failover Setting ---")
	if len(config.Atlasnet.AlternativeEnpoint.HostList) <= 0 {
		atlasnetFailover = NewRPCFailover([]RPCEndpoint{{
			Endpoint: "https://api.atlasnet.staratlas.cloud",
			Piority:  1,
			MaxRetry: 30}})
	} else {
		atlasnetFailover = NewRPCFailover(config.Atlasnet.AlternativeEnpoint.HostList)
	}

}

func main() {
	defer func() {
		if influxdb != nil {
			influxdb.ClientClose()
		}
		if database != nil {
			sqldb, err := database.DB()

			if err == nil {
				sqldb.Close()
			}
		}
	}()
	flag.Parse()
	go launchWorkers(RunAtlasnet)
	go APIService(RunAtlasnet)

	for {
		time.Sleep(10 * time.Second)
	}
}
