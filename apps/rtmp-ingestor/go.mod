module github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor

go 1.23.2

replace github.com/Harshitk-cp/streamhive/libs/proto => ../../libs/proto

require (
	github.com/Harshitk-cp/streamhive/libs/proto v0.0.0-00010101000000-000000000000
	github.com/nareix/joy4 v0.0.0-20200507095837-05a4ffbb5369
	github.com/prometheus/client_golang v1.22.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250428153025-10db94c68c34 // indirect
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/sys v0.32.0 // indirect
	google.golang.org/grpc v1.72.0
	google.golang.org/protobuf v1.36.6
)
