# distributed-tracing

## gRPC protocols

### [Prerequisites](https://grpc.io/docs/languages/go/quickstart/#prerequisites)

### Generating Go gRPC code

```shell
# generate gRPC golang code for common protocol
$ make generate-proto PROTOFILE=common/common.proto

# generate gRPC golang code for orders service protocol
$ make generate-proto PROTOFILE=orders/protocol/service.proto

# generate gRPC golang code for products service protocol
$ make generate-proto PROTOFILE=products/protocol/service.proto
```
