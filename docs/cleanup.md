# Cleanup commands

## Generated outputs
```bash
make clean-generated
# or:
rm -rf src/backend/bin src/flutter/build
```

## Containers
```bash
make clean-containers
# or:
docker compose -f docker-compose.yml down -v
```

Only stop or remove containers that belong to the Roomies stack.
