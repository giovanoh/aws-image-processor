package processor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nfnt/resize"
)

const (
	ThumbnailWidth  = 200
	ThumbnailHeight = 200
	MediumWidth     = 800
	MediumHeight    = 600
	JPEGQuality     = 85
)

type ProcessResult struct {
	ThumbnailKey string
	MediumKey    string
	OriginalKey  string
}

type ImageProcessor struct {
	s3Client     *s3.Client
	outputBucket string
}

func NewImageProcessor(cfg aws.Config) *ImageProcessor {
	// Ler bucket de saída da variável de ambiente
	outputBucket := os.Getenv("OUTPUT_BUCKET")
	if outputBucket == "" {
		// Fallback para valor padrão se não configurado
		outputBucket = "image-processor-out"
		log.Printf("OUTPUT_BUCKET não configurado, usando padrão: %s", outputBucket)
	}

	return &ImageProcessor{
		s3Client:     s3.NewFromConfig(cfg),
		outputBucket: outputBucket,
	}
}

func (p *ImageProcessor) ProcessImage(ctx context.Context, bucket, key string) (*ProcessResult, error) {
	log.Printf("Iniciando processamento de %s/%s", bucket, key)

	// 1. Baixar imagem original do S3
	img, contentType, err := p.downloadImage(ctx, bucket, key)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar imagem: %w", err)
	}

	// 2. Criar thumbnail
	thumbnailKey := fmt.Sprintf("thumbnails/%s", key)
	if err := p.createAndUploadVariant(ctx, img, thumbnailKey, ThumbnailWidth, ThumbnailHeight, contentType); err != nil {
		return nil, fmt.Errorf("erro ao criar thumbnail: %w", err)
	}

	// 3. Criar versão média
	mediumKey := fmt.Sprintf("medium/%s", key)
	if err := p.createAndUploadVariant(ctx, img, mediumKey, MediumWidth, MediumHeight, contentType); err != nil {
		return nil, fmt.Errorf("erro ao criar versão média: %w", err)
	}

	return &ProcessResult{
		ThumbnailKey: thumbnailKey,
		MediumKey:    mediumKey,
		OriginalKey:  key,
	}, nil
}

func (p *ImageProcessor) downloadImage(ctx context.Context, bucket, key string) (image.Image, string, error) {
	log.Printf("Baixando imagem...")

	result, err := p.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	defer result.Body.Close()

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	// Decodificar imagem baseado na extensão/content-type
	ext := strings.ToLower(filepath.Ext(key))
	var img image.Image

	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(result.Body)
	case ".png":
		img, err = png.Decode(result.Body)
	default:
		// Tentar decodificar automaticamente
		img, _, err = image.Decode(result.Body)
	}

	if err != nil {
		return nil, "", fmt.Errorf("erro ao decodificar imagem: %w", err)
	}

	log.Printf("Imagem baixada: %dx%d pixels", img.Bounds().Dx(), img.Bounds().Dy())
	return img, contentType, nil
}

func (p *ImageProcessor) createAndUploadVariant(
	ctx context.Context,
	img image.Image,
	key string,
	width, height uint,
	contentType string,
) error {
	log.Printf("Criando variante %dx%d: %s", width, height, key)

	// Redimensionar mantendo aspect ratio
	resized := resize.Thumbnail(width, height, img, resize.Lanczos3)

	// Codificar para bytes
	buf := new(bytes.Buffer)
	var err error

	if strings.Contains(contentType, "png") {
		err = png.Encode(buf, resized)
	} else {
		// Default para JPEG
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: JPEGQuality})
		contentType = "image/jpeg"
	}

	if err != nil {
		return fmt.Errorf("erro ao codificar imagem: %w", err)
	}

	// Upload para S3 (usando bucket da variável de ambiente)
	_, err = p.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.outputBucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
		//		ACL:         types.ObjectCannedACLPublicRead, // Opcional: tornar público
	})

	if err != nil {
		return fmt.Errorf("erro ao fazer upload: %w", err)
	}

	log.Printf("Variante criada: %s (%d bytes)", key, buf.Len())
	return nil
}
