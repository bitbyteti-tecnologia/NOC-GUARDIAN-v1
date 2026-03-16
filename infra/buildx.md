# Buildx multi-arch

# Criar builder
docker buildx create --name guardian-builder --use
docker buildx inspect --bootstrap

# Central
docker buildx build --platform linux/amd64,linux/arm64 -t nocguardian/central:latest -f central/Dockerfile central --push

# UI
docker buildx build --platform linux/amd64,linux/arm64 -t nocguardian/ui:latest -f UI/Dockerfile UI --push

# Proxy (incluindo arm/v7 e arm/v6 para Raspberrys antigos)
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7,linux/arm/v6 -t nocguardian/proxy:latest -f proxy/Dockerfile proxy --push
