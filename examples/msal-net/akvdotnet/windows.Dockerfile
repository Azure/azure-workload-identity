FROM mcr.microsoft.com/dotnet/runtime:5.0-nanoserver-1809
WORKDIR /app
COPY ./bin/release/netcoreapp5.0/publish/ .
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
ENTRYPOINT ["dotnet", "akvdotnet.dll"]
