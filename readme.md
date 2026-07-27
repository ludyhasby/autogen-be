## Pull Dependencies
go mod tidy

## Build Docker (for OTel, Jaeger, elasticSearch)
docker compose up -d

## Generator Swaggo
swag init -g ./cmd/web/main.go  --parseDependency --parseInternal