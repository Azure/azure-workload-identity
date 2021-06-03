FROM mcr.microsoft.com/dotnet/runtime:5.0
WORKDIR /app
COPY ./bin/release/netcoreapp5.0/publish/ .

ENTRYPOINT ["dotnet", "akvdotnet.dll"]
