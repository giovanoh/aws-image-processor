package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/giovanoh/aws-image-processor/internal/processor"
)

// S3EventRecord representa o evento do S3 dentro da mensagem SQS
type S3EventRecord struct {
	EventName string `json:"eventName"`
	S3        struct {
		Bucket struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key  string `json:"key"`
			Size int64  `json:"size"`
		} `json:"object"`
	} `json:"s3"`
}

type S3Event struct {
	Records []S3EventRecord `json:"Records"`
}

type Handler struct {
	processor *processor.ImageProcessor
}

func NewHandler(ctx context.Context) (*Handler, error) {
	// Carregar configuração AWS (credenciais, região, etc)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar config AWS: %w", err)
	}

	return &Handler{
		processor: processor.NewImageProcessor(cfg),
	}, nil
}

// HandleSQSEvent processa mensagens do SQS que contêm eventos do S3
// Retorna events.SQSEventResponse para reportar individualmente as falhas
func (h *Handler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	log.Printf("Recebidas %d mensagens do SQS", len(sqsEvent.Records))

	log.Printf("Evento recebido: %+v", sqsEvent)
	var batchItemFailures []events.SQSBatchItemFailure

	for i, record := range sqsEvent.Records {
		log.Printf("Processando mensagem %d/%d (MessageId: %s)",
			i+1, len(sqsEvent.Records), record.MessageId)

		// Parse do evento S3 que está no body da mensagem SQS
		var s3Event S3Event
		if err := json.Unmarshal([]byte(record.Body), &s3Event); err != nil {
			log.Printf("Erro ao parsear evento S3: %v", err)
			// Adiciona esta mensagem às falhas
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
			continue // Continua processando as outras mensagens
		}

		log.Printf("Evento S3 parseado: %+v", s3Event)

		// Processar cada objeto S3 mencionado no evento
		hasError := false
		for _, s3Record := range s3Event.Records {
			if err := h.processS3Object(ctx, s3Record); err != nil {
				log.Printf("Erro ao processar objeto: %v", err)
				hasError = true
				break
			}
		}

		// Se houve erro, reporta essa mensagem como falha
		if hasError {
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	if len(batchItemFailures) > 0 {
		log.Printf("%d mensagem(ns) falharam e serão retentadas", len(batchItemFailures))
	} else {
		log.Printf("Todas as %d mensagens processadas com sucesso", len(sqsEvent.Records))
	}

	// Retorna a resposta com as falhas (se houver)
	return events.SQSEventResponse{
		BatchItemFailures: batchItemFailures,
	}, nil
}

func (h *Handler) processS3Object(ctx context.Context, record S3EventRecord) error {
	bucket := record.S3.Bucket.Name
	key := record.S3.Object.Key
	size := record.S3.Object.Size

	log.Printf("Bucket: %s", bucket)
	log.Printf("Key: %s", key)
	log.Printf("Size: %d bytes (%.2f KB)", size, float64(size)/1024)

	// Validações
	if size > 50*1024*1024 { // 50 MB
		return fmt.Errorf("arquivo muito grande: %d bytes (máximo 50MB)", size)
	}

	// Processar imagem
	result, err := h.processor.ProcessImage(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("erro ao processar imagem: %w", err)
	}

	log.Printf("Imagem processada com sucesso!")
	log.Printf("   Thumbnail: %s", result.ThumbnailKey)
	log.Printf("   Medium: %s", result.MediumKey)
	log.Printf("   Original preservado: %s", key)

	return nil
}

func main() {
	ctx := context.Background()

	// Criar handler com configuração AWS
	handler, err := NewHandler(ctx)
	if err != nil {
		log.Fatalf("Erro ao criar handler: %v", err)
	}

	// Iniciar Lambda
	log.Println("Lambda iniciada e aguardando eventos...")
	lambda.Start(handler.HandleSQSEvent)
}
