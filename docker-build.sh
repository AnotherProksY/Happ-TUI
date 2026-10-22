docker build -t happ-tui-builder .
docker run --rm -v "$PWD:/out" happ-tui-builder sh -c "cp /src/happ-tui /src/happ-tui-core /out"
