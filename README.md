# Currency Rate Calculator

Currency Rate Calculator converts amount of base currency to quote currency

## Requirements

```bash
go v1.22
postgresql
docker
make (optional)
```

## How to run

- To run program install `docker/docker-compose` - needed to run postgresql container
- Configure your port in `docker-compose.yml` for postgres service
- After run:

```bash
docker-compose up -d
```

- Create API KEY for fast forex (I have put my own, you are welcome to use it!)
- Preconfigure your `.env` file
- Build or Run application

```bash
go run ./cmd/exchange-rate-calculator/main.go
```

- OR

```bash
make run
```
