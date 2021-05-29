FROM mcr.microsoft.com/dotnet/sdk:5.0 AS buildImg
WORKDIR /src
COPY . .
RUN mkdir app
RUN dotnet restore "{{ .csprojPath }}"
RUN dotnet build "{{ .csprojPath }}" -c Release -o /src/app
RUN dotnet publish "{{ .csprojPath }}" -c Release -o /src/app
FROM mcr.microsoft.com/dotnet/aspnet:5.0 AS runImg
WORKDIR /app
{{- range .ports}}
EXPOSE {{.}}
{{- end}}
ENV ASPNETCORE_URLS=http://+:{{ .httpPort }}
COPY --from=buildImg /src/app .
ENTRYPOINT ["dotnet", "{{ .appName }}.dll"]
