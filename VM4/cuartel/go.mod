module cuartel

go 1.27.0

require (
	github.com/go-sql-driver/mysql v1.10.1
	google.golang.org/grpc v1.85.0
	lab3-shared v0.0.0
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260817212433-ac3dfec99bb1 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace lab3-shared => ../../shared
