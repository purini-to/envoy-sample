# envoy-sample

## Run

```bash
docker run --rm -d -p 10000:10000 -v `pwa`/envoy.yaml:/etc/envoy/envoy.yaml envoyproxy/envoy-dev:6a6e43a94201e9059de61fd6e94fda984615b54c
```