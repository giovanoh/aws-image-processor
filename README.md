# Image Processing with S3 + SQS + Lambda

> :globe_with_meridians: Read this in other languages: [Português (Brasil)](README.pt-br.md)

## 📋 Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [Building the Project](#building-the-project)
- [AWS Console Configuration](#aws-console-configuration)
- [Testing the System](#testing-the-system)
- [Resource Cleanup](#resource-cleanup)
- [Next Steps](#next-steps)

---

## 🎯 Overview {#overview}

This project demonstrates a complete serverless architecture on AWS for automatic image processing. When an image is uploaded to S3, it is automatically processed (resizing, optimization) using Lambda Functions.

### Features
- Image upload to S3
- Asynchronous processing via Lambda
- Automatic thumbnail generation
- Automatic retry on failure (via SQS)
- Dead Letter Queue (DLQ) for problematic messages
- Logs on CloudWatch

---

## 🏗️ Architecture {#architecture}

### Data Flow
1. **Upload**: User uploads `photo.jpg` to bucket `image-processor-in`
2. **Event**: S3 sends notification to SQS queue `product-image-queue`
3. **Trigger**: Lambda is triggered automatically by SQS
4. **Processing**: Lambda downloads image, creates thumbnail and optimized versions
5. **Result**: Lambda saves processed images to bucket `image-processor-out`
6. **Retry**: If it fails, SQS retries automatically (up to 3 times)
7. **DLQ**: Messages that failed 3 times go to `product-image-queue-dlq`

### Detailed Flow Diagram

![Complete Flow Diagram](https://raw.githubusercontent.com/giovanoh/aws-image-processor/main/docs/architecture-flow.svg)

### Error Handling Flow

For a detailed sequence diagram showing the error handling and retry mechanism, see: [Error Flow Diagram](https://raw.githubusercontent.com/giovanoh/aws-image-processor/main/docs/architecture-flow-error.svg)

---

## 📦 Prerequisites {#prerequisites}

### Required Tools
- Go (Golang) ([Installation](https://go.dev/doc/install))
- AWS CLI ([Installation](https://aws.amazon.com/cli/))
- AWS Account (Free Tier works!)

### Configure AWS CLI
```bash
# Configure credentials
aws configure
# AWS Access Key ID: AKIAIOSFODNN7EXAMPLE
# AWS Secret Access Key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
# Default region name: us-east-1
# Default output format: json

# Test
aws s3 ls
```

---

## 📁 Project Structure {#project-structure}

```
aws-image-processor/
├── cmd/
│   └── lambda/
│       └── main.go                        # Lambda function entry point
├── internal/
│   ├── aws/
│   │   └── s3_client.go                   # Client for S3 operations
│   └── processor/
│       └── image_processor.go             # Main image processing logic
├── scripts/
│   └── build.sh                           # Build script
├── docs/
│   ├── architecture-flow.puml             # PlantUML architecture diagram
│   ├── architecture-flow.pt-br.puml       # PlantUML architecture diagram (Portuguese)
│   ├── architecture-flow-error.puml       # Error flow diagram
│   └── architecture-flow-error.pt-br.puml # Error flow diagram (Portuguese)
├── go.mod                                 # Go project dependencies
├── go.sum                                 # Dependency checksum
├── .gitignore                             # Files ignored by Git
├── README.md                              # English project documentation
└── README.pt-br.md                        # Brazilian Portuguese project documentation
```

---

## 🔨 Building the Project {#building-the-project}

Before configuring the AWS resources, you need to build the Lambda function. This project includes a build script that automates the entire process.

### Using the Build Script

The build script (`scripts/build.sh`) will:
1. Compile the Go code for Linux (required for Lambda)
2. Create a ZIP file with the compiled binary
3. Leave the artifact `lambda.zip` at the project root

**To build the project:**

```bash
# Make the script executable (first time only)
chmod +x scripts/build.sh

# Run the build script
./scripts/build.sh
```

**After running the script:**
- The `lambda.zip` file will be created at the root of the project
- This ZIP file is the artifact that you'll upload to AWS Lambda (see Step 3 in AWS Console Configuration)

**Prerequisites for building:**
- Go must be installed
- The `zip` utility must be installed (install with: `sudo apt install zip unzip` on Ubuntu/Debian)

---

## 🖥️ AWS Console Configuration {#aws-console-configuration}

### Step 1: Create S3 Buckets

> **📝 Note:** S3 bucket names must be globally unique. Add your name, initials or random number.

#### 1. Access S3 Console
   - Go to: https://console.aws.amazon.com/s3/
   - Click "Create bucket"

#### 2. Create Source Bucket (Original Images)

**Configuration:**
   - **Bucket name:** `image-processor-in-{your-name}` (ex: `image-processor-in-giovano`)
   - **Region:** `us-east-1` (or your preference)
   - **Block Public Access:**
     - ✅ Keep blocked (recommended for security)
   - **Versioning:** Enabled (optional, but recommended)
   - Click "Create bucket"

#### 3. Create Destination Bucket (Processed Images)

**Configuration:**
   - **Bucket name:** `image-processor-out-{your-name}` (ex: `image-processor-out-giovano`)
   - **Region:** Same region as source bucket (`us-east-1`)
   - **Block Public Access:**
     - ⚠️ Uncheck if you want to serve thumbnails publicly
     - ✅ OR keep blocked and use CloudFront later
   - Click "Create bucket"

**📌 Buckets Created:**
```
✅ Source Bucket:  image-processor-in-{your-name}
✅ Destination Bucket: image-processor-out-{your-name}
```

### Step 2: Create SQS Queue

#### 1. Access SQS Console
   - Go to: https://console.aws.amazon.com/sqs/
   - Click "Create queue"

#### 2. Configure Main Queue

**Configuration:**
   - **Name:** `product-image-queue`
   - **Type:** Standard
   - **Configuration:**
     - Visibility timeout: `300 seconds` (5 min)
     - Message retention period: `86400 seconds` (1 day)
     - Receive message wait time: `20 seconds` (long polling)
   - **Dead-letter queue:** (configure after creating DLQ)
   - Click "Create queue"

#### 3. Create Dead Letter Queue (DLQ)

**Configuration:**
   - **Name:** `product-image-queue-dlq`
   - **Type:** Standard
   - **Configuration:** Use defaults
   - Click "Create queue"

#### 4. Configure DLQ in Main Queue

   - Go back to queue `product-image-queue`
   - Click "Edit"
   - In section **Dead-letter queue:**
     - ✅ Enabled
     - Choose existing queue: `product-image-queue-dlq`
     - Maximum receives: `3`
   - Click "Save"

**📌 Queues Created:**
```
✅ Main Queue: product-image-queue
✅ Dead Letter Queue: product-image-queue-dlq
```

### Step 3: Create Lambda Function

1. **Access Lambda Console**
   - Go to: https://console.aws.amazon.com/lambda/
   - Click "Create function"

2. **Basic Configuration**
   - Option: **Author from scratch**
   - Function name: `ProcessImageFunction`
   - Runtime: **Amazon Linux 2023** (custom runtime)
   - Architecture: **x86_64**
   - Click "Create function"

3. **Upload Code**
   - In "Code source" section
   - Click "Upload from" → ".zip file"
   - Select `lambda.zip` (compiled binary)
   - Click "Save"

4. **Configure Handler**
   - In "Runtime settings" section → Edit
   - Handler: `bootstrap`

   > **💡 Explanation:** For custom runtime, the executable MUST be named "bootstrap".

5. **Function Settings**
   - **General configuration** → Edit:
     - Memory: `512 MB`
     - Timeout: `1 min`
     - Ephemeral storage: `512 MB`

   - **Environment variables** → Edit → Add environment variable:
     ```
     Key:   OUTPUT_BUCKET
     Value: image-processor-out-{your-name}
     ```

6. **Configure Permissions (IAM Role)**
   - In tab "Configuration" → "Permissions"
   - Click on Role name (will open IAM console)
   - "Add permissions" → "Attach policies"
   - Add:
     - ✅ `AmazonS3FullAccess`
     - ✅ `AWSLambdaSQSQueueExecutionRole`
     - ✅ `CloudWatchLogsFullAccess`

### Step 4: Connect S3 → SQS

#### 1. Configure SQS Permissions

Before creating the event notification, S3 needs permission to send messages to the queue:

1. Go to **SQS Console**
2. Select queue `product-image-queue`
3. Tab **"Access policy"** → **Edit**
4. **Replace** all content with the policy below:

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
      "Resource": "arn:aws:sqs:us-east-1:YOUR-ACCOUNT-ID:product-image-queue",
      "Condition": {
        "ArnEquals": {
          "aws:SourceArn": "arn:aws:s3:::image-processor-in-{your-name}"
        }
      }
    }
  ]
}
```

5. **Replace `YOUR-ACCOUNT-ID`** with your Account ID (12 digits)
6. Click **Save**

> **💡 How to find your Account ID:**
>
> **Option 1 - AWS Console:**
> - Click on your name in the upper right corner
> - The 12-digit number appears
>
> **Option 2 - AWS CLI:**
> ```bash
> aws sts get-caller-identity --query Account --output text
> ```

#### 2. Access S3 Bucket

   - Go to bucket `image-processor-in-{your-name}`
   - Tab "Properties"

#### 3. Event Notifications

   - Scroll to "Event notifications"
   - Click "Create event notification"

#### 4. Configure Event

**Configuration:**
   - **Event name:** `ImageUploadedEvent`
   - **Prefix:** *(leave empty to process all images)*
   - **Suffix:** `.jpg` (optional - only JPGs, or leave empty for all types)
   - **Event types:**
     - ✅ `s3:ObjectCreated:Put`
     - ✅ `s3:ObjectCreated:Post`
     - ✅ `s3:ObjectCreated:CompleteMultipartUpload`
   - **Destination:** SQS queue
   - **SQS queue:** `product-image-queue`
   - Click **"Save changes"**

**✅ If everything is correct, you will see a success message!**

### Step 5: Connect SQS → Lambda

#### 1. Access Lambda Function
   - Function: `ProcessImageFunction`

#### 2. Add Trigger

   - Click "Add trigger"
   - **Select a source:** SQS
   - **SQS queue:** `product-image-queue`
   - **Batch size:** `10` (processes up to 10 messages at a time)
   - **Batch window:** `5` seconds (optional)
   - Expand **"Additional settings"**:
     - ✅ **Report batch item failures** (IMPORTANT!)
   - ✅ **Enable trigger**
   - Click "Add"

#### 3. Verify Configuration

   - On the Lambda page, the diagram should appear:
   ```
   SQS (product-image-queue) → Lambda (ProcessImageFunction)
   ```

**📌 Complete Flow Configured:**
```
S3 (image-processor-in-{your-name})
   ↓ event
SQS (product-image-queue)
   ↓ trigger
Lambda (ProcessImageFunction)
   ↓ output
S3 (image-processor-out-{your-name})
```

---

## 🧪 Testing the System {#testing-the-system}

### 1. Manual Test via AWS CLI

```bash
# Upload a test image
aws s3 cp test-image.jpg s3://image-processor-in-{your-name}/test-image.jpg

# Check messages in queue
aws sqs get-queue-attributes \
    --queue-url https://sqs.us-east-1.amazonaws.com/ACCOUNT-ID/product-image-queue \
    --attribute-names ApproximateNumberOfMessages

# View Lambda logs
aws logs tail /aws/lambda/ProcessImageFunction --follow

# Check result in output bucket
aws s3 ls s3://image-processor-out-{your-name}/ --recursive

# View created thumbnails
aws s3 ls s3://image-processor-out-{your-name}/thumbnails/
aws s3 ls s3://image-processor-out-{your-name}/medium/
```

### 2. Test via Web Console

**Upload via S3 Console:**
1. Go to bucket `image-processor-in-{your-name}`
2. Click "Upload"
3. Drag a JPG image
4. Click "Upload"
5. Wait for processing (~5-10 seconds)
6. Check bucket `image-processor-out-{your-name}`
7. Should have folders `thumbnails/` and `medium/` with processed images

### 3. Monitoring in CloudWatch

```bash
# View logs in real time
aws logs tail /aws/lambda/ProcessImageFunction --follow --format short

# Search errors
aws logs filter-pattern "ERROR" \
    --log-group-name /aws/lambda/ProcessImageFunction \
    --start-time $(date -u -d '10 minutes ago' +%s)000

# Metrics
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

## 🧹 Resource Cleanup {#resource-cleanup}

### Why Clean Up?

Some AWS resources generate costs even when not in use:
- **S3:** Charges for storage (GB/month)
- **Lambda:** Usually free if not invoked
- **SQS:** Usually free on Free Tier
- **CloudWatch Logs:** Charges for log storage

### Correct Removal Order

It's important to follow the order to avoid dependency errors:

```
1. Disable Triggers and Event Notifications
2. Delete Lambda Function
3. Delete SQS Queues
4. Empty and Delete S3 Buckets
5. Delete IAM Roles and Policies (optional)
6. Delete CloudWatch Log Groups
```

---

## 🚀 Next Steps {#next-steps}

### Infrastructure as Code (IaC) with Terraform

Currently, this project requires manual configuration through the AWS Console. As a next step, Infrastructure as Code (IaC) can be implemented using Terraform to provision all AWS resources directly.

### CDN Configuration with CloudFront

To improve performance and reduce latency when serving processed images, a CloudFront distribution can be configured to deliver content from the output S3 bucket. This enables global caching and faster image delivery to end users.

---

**⭐ If this project was useful, consider giving it a star on GitHub!**
