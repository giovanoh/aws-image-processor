#!/bin/bash
set -e

# Verificar se zip está instalado
if ! command -v zip &> /dev/null; then
    echo "ERRO: 'zip' não está instalado."
    echo "Instale com: sudo apt install zip unzip"
    exit 1
fi

echo "Compilando Lambda para Linux..."

# Compilar para Linux (necessário para Lambda)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap -ldflags="-s -w" cmd/lambda/main.go

# Criar ZIP
echo "Criando arquivo ZIP..."
zip lambda.zip bootstrap

# Limpar
rm bootstrap

echo "Build concluído! Arquivo: lambda.zip"
ls -lh lambda.zip