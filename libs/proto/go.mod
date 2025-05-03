module github.com/Harshitk-cp/streamhive/libs/proto

go 1.23.2

require (
	google.golang.org/grpc v1.72.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250428153025-10db94c68c34 // indirect
)

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20230822172742-b8732ec3820d

// Ensure consistent version of googleapis/rpc
replace google.golang.org/genproto/googleapis/rpc => google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a
