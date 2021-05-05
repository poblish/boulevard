go test ./... -coverprofile=c.out && \
go tool cover -html=c.out && \
rm -f c.out