> go mod tidy 

```
Prune any no-longer-needed dependencies from go.mod and add any dependencies needed for other combinations of OS, architecture, and build tags
```

> go mod vendor

```
Optional step to create a vendor directory
```

> go build -o bin/main -v .