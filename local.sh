export SERVER_EMBEDDED=false
export MODEM_PORT=/dev/ttyUSB3
export SERVER_AUTH_CREDENTIALS_FILE=./local-credentials
go run main.go $@

