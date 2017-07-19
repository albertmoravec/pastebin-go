# Pastebin

Fancy pastebin written in GO and Redis.

## Features

* User defined expiration of pastes.
* Works with or without Javascript.
* Syntax highlighting while viewing or creating a paste (when JS is enabled).
* Simple API

![main](https://raw.githubusercontent.com/sp444/pastebin-go/master/assets/docs/main.png)

![save](https://raw.githubusercontent.com/sp444/pastebin-go/master/assets/docs/save.png)

### Demo

https://a.sp44.me

## Installation

### Requirements

* Golang
* Redis

### Compiling

```
go get github.com/sp444/pastebin
go get .
go build pastebin.go
./pastebin 
```

### Configuration

See config.json

#### Persistence

You should probably enable AOF persistence if you don't want to lose all your data when the Redis server goes offline for whatever reason. More info here https://redis.io/topics/persistence 

