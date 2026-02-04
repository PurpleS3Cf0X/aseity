# Docker Support

Aseity works great in containers, keeping your host system clean.

## Prerequisites
- Docker Desktop or Docker Engine installed.

## Using Docker Compose
We provide a `docker-compose.yml` that sets up:
1. **Ollama**: Running in a container (GPU enabled if on Linux/Windows, CPU/Metal on Mac).
2. **Aseity**: The client.

### Commands

**Start the Stack:**
```bash
make docker-up
```
This spins up Ollama in the background and drops you into the Aseity shell.

**Specific Profiles:**
If you want to use vLLM instead of Ollama:
```bash
make docker-up-vllm
```

**Cleanup:**
```bash
make docker-down
```
This stops and removes all containers.

## Manual Docker Run
If you just want to run the image:

```bash
docker build -t aseity .
docker run -it -v $(pwd):/app aseity
```
Note: You'll need to handle networking to reach a local Ollama instance (typically using `--network host` on Linux).

## Mac Optimization
Our Docker setup is optimized for Apple Silicon (M1/M2/M3). It uses the official Ollama image which supports Metal acceleration inside Docker, ensuring strictly local performance is still fast.
