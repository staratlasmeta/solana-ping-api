package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	// jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// ConnectionMode is a type of client connection
type ConnectionMode string

const (
	HTTP  ConnectionMode = "http"
	HTTPS ConnectionMode = "https"
	BOTH  ConnectionMode = "both"
)

type PingConfig struct {
	Receiver                string
	NumWorkers              int
	BatchCount              int
	BatchInverval           int
	TxTimeout               int64
	WaitConfirmationTimeout int64
	StatusCheckInterval     int64
	MinPerPingTime          int64
	ComputeFeeDualMode      bool
	RequestUnits            uint32
	ComputeUnitPrice        uint64
}
type WebHookConfig struct {
	Enabled bool
	Webhook string
}
type SlackReport struct {
	Report WebHookConfig
	Alert  WebHookConfig
}
type DiscordReport struct {
	BotName      string
	BotAvatarURL string
	Report       WebHookConfig
	Alert        WebHookConfig
}
type Report struct {
	Enabled       bool
	Interval      int
	LossThreshold float64
	LevelFilePath string
	Slack         SlackReport
	Discord       DiscordReport
}
type APIServer struct {
	Enabled bool
	Mode    ConnectionMode
	IP      string
	SSLIP   string
	KeyPath string
	CrtPath string
}
type Database struct {
	UseGoogleCloud       bool
	GCloudCredentialPath string
	DBConn               string
}
type InfluxdbConfig struct {
	Enabled     bool
	InfluxdbURL string
	AccessToken string
	Orgnization string
	Bucket      string
}
type Retension struct {
	Enabled           bool
	KeepHours         int64
	UpdateIntervalSec int64
}

type SolanaCLIConfig struct {
	JsonRPCURL    string
	WebsocketURL  string
	KeypairPath   string
	AddressLabels map[string]string
	Commitment    string
}

type ClusterCLIConfig struct {
	Dir            string
	MainnetPath    string
	TestnetPath    string
	DevnetPath     string
	AtlasnetPath   string
	ConfigMain     SolanaCLIConfig
	ConfigTestnet  SolanaCLIConfig
	ConfigDevnet   SolanaCLIConfig
	ConfigAtlasnet SolanaCLIConfig
}

type RPCEndpoint struct {
	Endpoint    string
	AccessToken string
	Piority     int
	MaxRetry    int
}
type EndpointAlert struct {
	Enabled bool
	Webhook string
}
type AlternativeEnpoint struct {
	HostList     []RPCEndpoint
	SlackAlert   EndpointAlert
	DiscordAlert EndpointAlert
}

type ClusterPing struct {
	APIServer
	PingServiceEnabled bool
	AlternativeEnpoint
	PingConfig
	Report
}

type ClusterConfig struct {
	Cluster
	HostName string
	ClusterPing
}

type Config struct {
	Database
	InfluxdbConfig
	Mainnet  ClusterConfig
	Atlasnet ClusterConfig
	Testnet  ClusterConfig
	Devnet   ClusterConfig
	ClusterCLIConfig
	Retension
}

func loadConfig() Config {
	// jww.SetLogThreshold(jww.LevelTrace)
	// jww.SetStdoutThreshold(jww.LevelTrace)
	c := Config{}
	v := viper.New()

	v.AddConfigPath("./")
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	// setup config.yaml
	v.SetConfigName("config")
	v.ReadInConfig()
	// setup config.yaml (Database)
	c.Database.UseGoogleCloud = v.GetBool("Database.UseGoogleCloud")
	c.Database.GCloudCredentialPath = v.GetString("Database.GCloudCredentialPath")
	c.DBConn = v.GetString("Database.DBConn")
	gcloudCredential := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if len(gcloudCredential) == 0 && len(c.Database.GCloudCredentialPath) != 0 {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", c.Database.GCloudCredentialPath)
	}
	log.Println("GOOGLE_APPLICATION_CREDENTIALS=", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	// setup influxdb in config.yaml
	c.InfluxdbConfig = InfluxdbConfig{
		Enabled:     v.GetBool("InfluxdbConfig.Enabled"),
		InfluxdbURL: v.GetString("InfluxdbConfig.InfluxdbURL"),
		AccessToken: v.GetString("InfluxdbConfig.AccessToken"),
		Orgnization: v.GetString("InfluxdbConfig.Orgnization"),
		Bucket:      v.GetString("InfluxdbConfig.Bucket"),
	}
	// setup config.yaml (Retension)
	c.Retension = Retension{
		Enabled:           v.GetBool("Retension.Enabled"),
		KeepHours:         v.GetInt64("Retension.KeepHours"),
		UpdateIntervalSec: v.GetInt64("Retension.UpdateIntervalSec"),
	}
	// setup config.yaml (ClusterConfigFile)
	c.ClusterCLIConfig = ClusterCLIConfig{
		Dir:          v.GetString("SolanaCliFile.Dir"),
		AtlasnetPath: v.GetString("SolanaCliFile.AtlasnetPath"),
	}

	if len(c.ClusterCLIConfig.AtlasnetPath) > 0 {
		sConfig, err := ReadSolanaCLIConfigFile(c.ClusterCLIConfig.Dir + c.ClusterCLIConfig.AtlasnetPath)
		if err != nil {
			log.Fatal(err)
		}
		c.ClusterCLIConfig.ConfigAtlasnet = sConfig
	}
	// setup  config.yaml (SolanaCliFile) all cluster services
	configAtlasnetFile := v.GetString("ClusterConfigFile.Atlasnet")
	// Read Each Cluster Configurations
	// setup config.yaml for mainnet
	v.SetConfigName(configAtlasnetFile)
	v.ReadInConfig()
	c.Atlasnet = ClusterConfig{
		Cluster:     Atlasnet,
		HostName:    hostname,
		ClusterPing: ReadClusterPingConfig(v),
	}
	if c.Mainnet.APIServer.Mode != HTTP &&
		c.Mainnet.APIServer.Mode != HTTPS && c.Mainnet.APIServer.Mode != BOTH {
		c.Mainnet.APIServer.Mode = HTTP
		log.Println("Mainnet API server mode not support! use default mode")
	}
	return c
}

func ReadSolanaCLIConfigFile(filepath string) (SolanaCLIConfig, error) {
	configmap := make(map[string]string, 1)
	addressmap := make(map[string]string, 1)

	f, err := os.Open(filepath)
	if err != nil {
		log.Printf("error opening file: %v\n", err)
		return SolanaCLIConfig{}, err
	}
	r := bufio.NewReader(f)
	line, _, err := r.ReadLine()
	for err == nil {
		k, v := ToKeyPair(string(line))
		if k == "address_labels" {
			line, _, err := r.ReadLine()
			lKey, lVal := ToKeyPair(string(line))
			if err == nil && string(line)[0:1] == " " {
				if len(lKey) > 0 && len(lVal) > 0 {
					addressmap[lKey] = lVal
				}
			} else {
				configmap[k] = v
			}
		} else {
			configmap[k] = v
		}

		line, _, err = r.ReadLine()
	}
	return SolanaCLIConfig{
		JsonRPCURL:    configmap["json_rpc_url"],
		WebsocketURL:  configmap["websocket_url:"],
		KeypairPath:   configmap["keypair_path"],
		AddressLabels: addressmap,
		Commitment:    configmap["commitment"],
	}, nil
}

func ToKeyPair(line string) (key string, val string) {
	noSpaceLine := strings.TrimSpace(string(line))
	idx := strings.Index(noSpaceLine, ":")
	if idx == -1 || idx == 0 { // not found or only have :
		return "", ""
	}
	if (len(noSpaceLine) - 1) == idx { // no value
		return strings.TrimSpace(noSpaceLine[0:idx]), ""
	}
	return strings.TrimSpace(noSpaceLine[0:idx]), strings.TrimSpace(noSpaceLine[idx+1:])
}

func ReadClusterPingConfig(v *viper.Viper) ClusterPing {
	v.Debug()
	clusterConf := ClusterPing{}
	v.Unmarshal(&clusterConf)
	return clusterConf
}
