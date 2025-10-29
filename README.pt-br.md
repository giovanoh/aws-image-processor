# Processamento de Imagens com S3 + SQS + Lambda

> :globe_with_meridians: Leia em outros idiomas: [English](README.md)

## 📋 Índice
- [Visão Geral](#visão-geral)
- [Arquitetura](#arquitetura)
- [Pré-requisitos](#pré-requisitos)
- [Estrutura do Projeto](#estrutura-do-projeto)
- [Build do Projeto](#build-do-projeto)
- [Configuração AWS Console](#configuração-aws-console)
- [Testando o Sistema](#testando-o-sistema)
- [Troubleshooting](#troubleshooting)
- [Limpeza de Recursos](#limpeza-de-recursos-importante)
- [Próximos Passos](#próximos-passos)

---

## 🎯 Visão Geral

Este projeto demonstra uma arquitetura serverless completa na AWS para processamento automático de imagens. Quando uma imagem é enviada ao S3, ela é automaticamente processada (redimensionamento, otimização) usando Lambda Functions.

### Funcionalidades
- Upload de imagens para S3
- Processamento assíncrono via Lambda
- Geração automática de thumbnails
- Retry automático em caso de falha (via SQS)
- Dead Letter Queue (DLQ) para mensagens problemáticas
- Logs no CloudWatch

---

## 🏗️ Arquitetura

### Fluxo de Dados
1. **Upload**: Usuário faz upload de `foto.jpg` para bucket `image-processor-in`
2. **Evento**: S3 envia notificação para fila SQS `product-image-queue`
3. **Trigger**: Lambda é acionada automaticamente pelo SQS
4. **Processamento**: Lambda baixa imagem, cria thumbnail e versões otimizadas
5. **Resultado**: Lambda salva imagens processadas no bucket `image-processor-out`
6. **Retry**: Se falhar, SQS retenta automaticamente (até 3 vezes)
7. **DLQ**: Mensagens que falharam 3x vão para `product-image-queue-dlq`

### Diagrama Detalhado

![Diagrama de Fluxo Completo](docs/Image%20Processor%20-%20Fluxo%20Completo.svg)

### Diagrama de Tratamento de Erros

Para um diagrama de sequência detalhado mostrando o tratamento de erros e mecanismo de retry, veja: [Diagrama de Fluxo de Erro](docs/Image%20Processor%20-%20Fluxo%20com%20Erro%20e%20Retry.svg)

---

## 📦 Pré-requisitos

### Ferramentas Necessárias
- Go (Golang) ([Instalação](https://go.dev/doc/install))
- AWS CLI ([Instalação](https://aws.amazon.com/cli/))
- Conta AWS (Free Tier funciona!)

### Configurar AWS CLI
```bash
# Configurar credenciais
aws configure
# AWS Access Key ID: AKIAIOSFODNN7EXAMPLE
# AWS Secret Access Key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
# Default region name: us-east-1
# Default output format: json

# Testar
aws s3 ls
```

---

## 📁 Estrutura do Projeto

```
aws-image-processor/
├── cmd/
│   └── lambda/
│       └── main.go                    # Ponto de entrada da função Lambda
├── internal/
│   ├── aws/
│   │   └── s3_client.go               # Cliente para operações S3
│   └── processor/
│       └── image_processor.go         # Lógica principal de processamento de imagens
├── scripts/
│   └── build.sh                       # Script de build
├── docs/
│   ├── architecture-flow.puml         # Diagrama da arquitetura em PlantUML
│   └── architecture-flow-error.puml   # Diagrama de fluxo de erro
├── go.mod                             # Dependências do projeto Go
├── go.sum                             # Checksum das dependências
├── .gitignore                         # Arquivos ignorados pelo Git
├── README.md                          # Documentação do projeto em inglês
└── README.pt-br.md                    # Documentação do projeto em português
```

---

## 🔨 Build do Projeto

Antes de configurar os recursos AWS, é necessário fazer o build da função Lambda. Este projeto inclui um script de build que automatiza todo o processo.

### Usando o Script de Build

O script de build (`scripts/build.sh`) vai:
1. Compilar o código Go para Linux (necessário para Lambda)
2. Criar um arquivo ZIP com o binário compilado
3. Deixar o artefato `lambda.zip` na raiz do projeto

**Para fazer o build do projeto:**

```bash
# Tornar o script executável (apenas na primeira vez)
chmod +x scripts/build.sh

# Executar o script de build
./scripts/build.sh
```

**Após executar o script:**
- O arquivo `lambda.zip` será criado na raiz do projeto
- Este arquivo ZIP é o artefato que você vai fazer upload para o AWS Lambda (veja Passo 3 em Configuração AWS Console)

**Pré-requisitos para o build:**
- Go deve estar instalado
- O utilitário `zip` deve estar instalado (instale com: `sudo apt install zip unzip` no Ubuntu/Debian)

---

## 🖥️ Configuração AWS Console

### Passo 1: Criar Buckets S3

> **📝 Nota:** Nomes de buckets S3 devem ser únicos globalmente. Adicione seu nome, iniciais ou número aleatório.

#### 1. Acessar S3 Console
   - Ir para: https://console.aws.amazon.com/s3/
   - Clicar em "Create bucket"

#### 2. Criar Bucket de Origem (Imagens Originais)

**Configuração:**
   - **Bucket name:** `image-processor-in-{seu-nome}` (ex: `image-processor-in-giovano`)
   - **Region:** `us-east-1` (ou sua preferência)
   - **Block Public Access:**
     - ✅ Manter bloqueado (recomendado para segurança)
   - **Versioning:** Enabled (opcional, mas recomendado)
   - Clicar em "Create bucket"

#### 3. Criar Bucket de Destino (Imagens Processadas)

**Configuração:**
   - **Bucket name:** `image-processor-out-{seu-nome}` (ex: `image-processor-out-giovano`)
   - **Region:** Mesma região do bucket de origem (`us-east-1`)
   - **Block Public Access:**
     - ⚠️ Desmarcar se quiser servir thumbnails publicamente
     - ✅ OU manter bloqueado e usar CloudFront depois
   - Clicar em "Create bucket"

**📌 Buckets Criados:**
```
✅ Bucket de Origem:  image-processor-in-{seu-nome}
✅ Bucket de Destino: image-processor-out-{seu-nome}
```

### Passo 2: Criar Fila SQS

#### 1. Acessar SQS Console
   - Ir para: https://console.aws.amazon.com/sqs/
   - Clicar em "Create queue"

#### 2. Configurar Fila Principal

**Configuração:**
   - **Name:** `product-image-queue`
   - **Type:** Standard
   - **Configuration:**
     - Visibility timeout: `300 seconds` (5 min)
     - Message retention period: `86400 seconds` (1 dia)
     - Receive message wait time: `20 seconds` (long polling)
   - **Dead-letter queue:** (configurar depois de criar a DLQ)
   - Clicar em "Create queue"

#### 3. Criar Dead Letter Queue (DLQ)

**Configuração:**
   - **Name:** `product-image-queue-dlq`
   - **Type:** Standard
   - **Configuration:** Usar padrões
   - Clicar em "Create queue"

#### 4. Configurar DLQ na Fila Principal

   - Voltar para fila `product-image-queue`
   - Clicar em "Edit"
   - Na seção **Dead-letter queue:**
     - ✅ Enabled
     - Choose existing queue: `product-image-queue-dlq`
     - Maximum receives: `3`
   - Clicar em "Save"

**📌 Filas Criadas:**
```
✅ Fila Principal: product-image-queue
✅ Dead Letter Queue: product-image-queue-dlq
```

### Passo 3: Criar Função Lambda

1. **Acessar Lambda Console**
   - Ir para: https://console.aws.amazon.com/lambda/
   - Clicar em "Create function"

2. **Configuração Básica**
   - Option: **Author from scratch**
   - Function name: `ProcessImageFunction`
   - Runtime: **Amazon Linux 2023** (custom runtime)
   - Architecture: **x86_64**
   - Clicar em "Create function"

   > **⚠️ Importante:** Use "provided.al2023" com binário customizado.

3. **Upload do Código**
   - Na seção "Code source"
   - Clicar em "Upload from" → ".zip file"
   - Selecionar `lambda.zip` (binário compilado)
   - Clicar em "Save"

4. **Configurar Handler**
   - Na seção "Runtime settings" → Edit
   - Handler: `bootstrap`

   > **💡 Explicação:** Para custom runtime, o executável DEVE se chamar "bootstrap".

5. **Configurações da Função**
   - **General configuration** → Edit:
     - Memory: `512 MB`
     - Timeout: `1 min`
     - Ephemeral storage: `512 MB`

   - **Environment variables** → Edit → Add environment variable:
     ```
     Key:   OUTPUT_BUCKET
     Value: image-processor-out-{seu-nome}
     ```

6. **Configurar Permissões (IAM Role)**
   - Na aba "Configuration" → "Permissions"
   - Clicar no Role name (vai abrir IAM console)
   - "Add permissions" → "Attach policies"
   - Adicionar:
     - ✅ `AmazonS3FullAccess`
     - ✅ `AWSLambdaSQSQueueExecutionRole`
     - ✅ `CloudWatchLogsFullAccess`

### Passo 4: Conectar S3 → SQS

#### 1. Configurar Permissões SQS

Antes de criar o event notification, o S3 precisa ter permissão para enviar mensagens para a fila:

1. Ir para **SQS Console**
2. Selecionar fila `product-image-queue`
3. Aba **"Access policy"** → **Edit**
4. **Substituir** todo o conteúdo pela política abaixo:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "s3.amazonaws.com"
      },
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:us-east-1:SEU-ACCOUNT-ID:product-image-queue",
      "Condition": {
        "ArnEquals": {
          "aws:SourceArn": "arn:aws:s3:::image-processor-in-{seu-nome}"
        }
      }
    }
  ]
}
```

5. **Substituir `SEU-ACCOUNT-ID`** pelo seu Account ID (12 dígitos)
6. Clicar em **Save**

> **💡 Como descobrir seu Account ID:**
>
> **Opção 1 - Console AWS:**
> - Clicar no seu nome no canto superior direito
> - Aparece o número de 12 dígitos
>
> **Opção 2 - AWS CLI:**
> ```bash
> aws sts get-caller-identity --query Account --output text
> ```

#### 2. Acessar Bucket S3

   - Ir para bucket `image-processor-in-{seu-nome}`
   - Aba "Properties"

#### 3. Event Notifications

   - Descer até "Event notifications"
   - Clicar em "Create event notification"

#### 4. Configurar Evento

**Configuração:**
   - **Event name:** `ImageUploadedEvent`
   - **Prefix:** *(deixe vazio para processar todas as imagens)*
   - **Suffix:** `.jpg` (opcional - apenas JPGs, ou deixe vazio para todos os tipos)
   - **Event types:**
     - ✅ `s3:ObjectCreated:Put`
     - ✅ `s3:ObjectCreated:Post`
     - ✅ `s3:ObjectCreated:CompleteMultipartUpload`
   - **Destination:** SQS queue
   - **SQS queue:** `product-image-queue`
   - Clicar em **"Save changes"**

**✅ Se tudo estiver correto, você verá uma mensagem de sucesso!**

### Passo 5: Conectar SQS → Lambda

#### 1. Acessar Lambda Function
   - Função: `ProcessImageFunction`

#### 2. Adicionar Trigger

   - Clicar em "Add trigger"
   - **Select a source:** SQS
   - **SQS queue:** `product-image-queue`
   - **Batch size:** `10` (processa até 10 mensagens por vez)
   - **Batch window:** `5` seconds (opcional)
   - Expandir **"Additional settings"**:
     - ✅ **Report batch item failures** (IMPORTANTE!)
   - ✅ **Enable trigger**
   - Clicar em "Add"

#### 3. Verificar Configuração

   - Na página da Lambda, deve aparecer o diagrama:
   ```
   SQS (product-image-queue) → Lambda (ProcessImageFunction)
   ```

**📌 Fluxo Completo Configurado:**
```
S3 (image-processor-in-{seu-nome})
   ↓ evento
SQS (product-image-queue)
   ↓ trigger
Lambda (ProcessImageFunction)
   ↓ output
S3 (image-processor-out-{seu-nome})
```

---

## 🧪 Testando o Sistema

### 1. Teste Manual via AWS CLI

```bash
# Upload de uma imagem de teste
aws s3 cp test-image.jpg s3://image-processor-in-{seu-nome}/test-image.jpg

# Verificar mensagens na fila
aws sqs get-queue-attributes \
    --queue-url https://sqs.us-east-1.amazonaws.com/ACCOUNT-ID/product-image-queue \
    --attribute-names ApproximateNumberOfMessages

# Ver logs da Lambda
aws logs tail /aws/lambda/ProcessImageFunction --follow

# Verificar resultado no bucket de saída
aws s3 ls s3://image-processor-out-{seu-nome}/ --recursive

# Ver thumbnails criados
aws s3 ls s3://image-processor-out-{seu-nome}/thumbnails/
aws s3 ls s3://image-processor-out-{seu-nome}/medium/
```

### 2. Teste via Console Web

**Upload via Console S3:**
1. Ir para bucket `image-processor-in-{seu-nome}`
2. Clicar em "Upload"
3. Arrastar uma imagem JPG
4. Clicar em "Upload"
5. Aguardar processamento (~5-10 segundos)
6. Verificar bucket `image-processor-out-{seu-nome}`
7. Deve ter pastas `thumbnails/` e `medium/` com as imagens processadas

### 3. Monitoramento no CloudWatch

```bash
# Ver logs em tempo real
aws logs tail /aws/lambda/ProcessImageFunction --follow --format short

# Buscar erros
aws logs filter-pattern "ERROR" \
    --log-group-name /aws/lambda/ProcessImageFunction \
    --start-time $(date -u -d '10 minutes ago' +%s)000

# Métricas
aws cloudwatch get-metric-statistics \
    --namespace AWS/Lambda \
    --metric-name Invocations \
    --dimensions Name=FunctionName,Value=ProcessImageFunction \
    --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
    --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
    --period 300 \
    --statistics Sum
```

---

## 🧹 Limpeza de Recursos

### Por que Limpar?

Alguns recursos AWS geram custos mesmo quando não estão em uso:
- **S3:** Cobra por armazenamento (GB/mês)
- **Lambda:** Geralmente grátis se não invocar
- **SQS:** Geralmente grátis no Free Tier
- **CloudWatch Logs:** Cobra por armazenamento de logs

### Ordem Correta de Remoção

É importante seguir a ordem para evitar erros de dependência:

```
1. Desabilitar Triggers e Event Notifications
2. Deletar Lambda Function
3. Deletar Filas SQS
4. Esvaziar e Deletar Buckets S3
5. Deletar IAM Roles e Policies (opcional)
6. Deletar CloudWatch Log Groups
```

---

## 🚀 Próximos Passos

### Infrastructure as Code (IaC) com Terraform

Atualmente, este projeto requer configuração manual através do Console AWS. Como próximo passo, pode ser implementado Infrastructure as Code (IaC) usando Terraform para provisionar todos os recursos AWS diretamente.

---

**⭐ Se este projeto foi útil, considere dar uma estrela no GitHub!**
