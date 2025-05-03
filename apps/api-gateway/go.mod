module github.com/Harshitk-cp/streamhive/apps/api-gateway

go 1.23.2

replace github.com/Harshitk-cp/streamhive/libs/proto => ../../libs/proto

require (
	github.com/Harshitk-cp/streamhive/libs/proto v0.0.0-00010101000000-000000000000
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	golang.org/x/time v0.11.0
	google.golang.org/grpc v1.72.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250428153025-10db94c68c34 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20230822172742-b8732ec3820d

// Ensure consistent version of googleapis/rpc
replace google.golang.org/genproto/googleapis/rpc => google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a
