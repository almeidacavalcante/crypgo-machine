# CrypGo Machine 🚀

Bot de trading de criptomoedas automatizado desenvolvido em Go com CI/CD automático.

## 🔥 Deploy Automático Ativo
Este projeto possui CI/CD automático com GitHub Actions para deploy na VPS de produção.

## High level design
```
app/
│
├── domain/
│   ├── entities/
│   │   └── trade_bot.go
│   ├── value_objects/
│   │   └── strategy.go
│   ├── repositories/
│   │   └── trade_bot_repository.go
│   └── services/
│       └── trading_service.go
│
├── application/
│   ├── usecases/
│   │   ├── create_trade_bot.go
│   │   ├── start_trade_bot.go
│   │   └── stop_trade_bot.go
│   └── dtos/
│       └── create_trade_bot_dto.go
│
├── infrastructure/
│   ├── database/
│   │   └── trade_bot_repository_mongo.go
│   ├── binance/
│   │   └── binance_client.go
│   └── events/
│       └── event_emitter.go
│
├── interfaces/
│   ├── http/
│   │   └── handlers/
│   │       └── trade_bot_handler.go
│   └── workers/
│       └── trade_bot_worker.go
│
└── main.go

```