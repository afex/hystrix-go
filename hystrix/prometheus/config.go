package hystrix_go

type HystrixConfig struct {
	CommandName            string `env:"COMMAND_NAME"`
	Timeout                int    `env:"TIMEOUT_IN_MS"`
	MaxConcurrentRequests  int    `env:"MAX_CONCURRENT_REQUESTS"`
	RequestVolumeThreshold int    `env:"REQUEST_VOLUME_THRESHOLD"`
	SleepWindow            int    `env:"SLEEP_WINDOW"`
	ErrorPercentThreshold  int    `env:"ERROR_PERCENT_THRESHOLD"`
}
