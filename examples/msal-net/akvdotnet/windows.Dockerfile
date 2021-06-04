FROM mcr.microsoft.com/dotnet/runtime:5.0-nanoserver-1809
WORKDIR /app
COPY ./bin/release/netcoreapp5.0/publish/ .

ENTRYPOINT ["dotnet", "akvdotnet.dll"]
