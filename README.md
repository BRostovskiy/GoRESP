# GoRESP(by `Boris Rostovskiy`)

## Simple implementation `redis` server for testing purposes
Tested in combination with clients:

* Python3 redis client
* Golang 'redigo' client (usage also showed in ./binaries/client)
* Official redis client

## For local deployment pls usr docker-compose(adjust the usage bind_port in `docker-compose.yaml` for your specific needs):
* Run `docker-compose up --build goredis` and then use any client(for example `redis-cli -p PORT`) you want to perform operations
* Run `docker exec -it goredis ./goredisclient` to see the outcome of client work(see more options like `key_val` and `getOnly`) 
 
## Supported operations(number of supported operations is limited to simple strings manipulations):
* `SET` - SET KEY and corresponding VALUE
* `GET` - GET KEY

## Future improvements:
* add more operations
* add more tests
