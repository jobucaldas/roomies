# Cleanup commands

## Generated outputs
- `make clean-generated`
- `rm -rf target src/backend/bin src/app/dist src/app/target`

## Containers / processes
- `make clean-containers`
- `podman ps --filter name=roomies`
- `podman-compose -f docker-compose.yml down -v`
- `podman-compose -f .devcontainer/docker-compose.yml down -v`

## Notes
Only stop or remove containers that belong to the Roomies stack.
