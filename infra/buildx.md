# Buildx multi-arch

# Criar builder
docker buildx create --name guardian-builder --use
docker buildx inspect --bootstrap

# Central
docker buildx build --platform linux/amd64,linux/arm64 -t nocguardian/central:latest -f central/deploy/Dockerfile central --push

# UI
docker buildx build --platform linux/amd64,linux/arm64 -t nocguardian/ui:latest -f ui/deploy/Dockerfile ui --push

# Proxy (incluindo arm/v7 e arm/v6 para Raspberrys antigos)
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7,linux/arm/v6 -t nocguardian/proxy:latest -f proxy/deploy/Dockerfile proxy --push
