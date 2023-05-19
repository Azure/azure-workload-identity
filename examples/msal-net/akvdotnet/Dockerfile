FROM --platform=linux/amd64 mcr.microsoft.com/dotnet/sdk:7.0 AS builder
WORKDIR /app
ADD . .
RUN dotnet build akvdotnet.csproj && dotnet publish -c release

ARG BASEIMAGE
FROM ${BASEIMAGE:-mcr.microsoft.com/dotnet/runtime:5.0}
WORKDIR /app
COPY --from=builder /app/bin/release/netcoreapp5.0/publish/ .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
ENTRYPOINT ["dotnet", "akvdotnet.dll"]
