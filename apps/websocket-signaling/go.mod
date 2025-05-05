// apps/websocket-signaling/go.mod
module github.com/Harshitk-cp/streamhive/apps/websocket-signaling

go 1.23.2

replace github.com/Harshitk-cp/streamhive/libs/proto => ../../libs/proto

require (
	github.com/Harshitk-cp/streamhive/libs/proto v0.0.0-20250505063951-8bbcb806e434
	github.com/gorilla/websocket v1.5.3
	github.com/prometheus/client_golang v1.22.0
	google.golang.org/grpc v1.72.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250428153025-10db94c68c34 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
